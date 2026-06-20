// Package cloudbackup uploads local backup archives to remote destinations
// (S3-compatible object storage, WebDAV, or SSH/SFTP) without third-party SDKs.
package cloudbackup

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// Destination describes where and how to upload a backup.
type Destination struct {
	Type      string // s3 | sftp | webdav
	Endpoint  string // S3 endpoint host (or full URL), SSH host:port, or WebDAV base URL
	Bucket    string // S3 bucket, or remote base directory
	AccessKey string // S3 access key / SSH user / WebDAV user
	SecretKey string // S3 secret / SSH password / WebDAV password
	Region    string // S3 region (default "us-east-1")
}

// Upload sends the file at localPath to the destination under the given object
// key (typically the archive's base name).
func Upload(ctx context.Context, d Destination, localPath, key string) error {
	switch d.Type {
	case "s3":
		return uploadS3(ctx, d, localPath, key)
	case "webdav":
		return uploadWebDAV(ctx, d, localPath, key)
	case "sftp":
		return uploadSFTP(ctx, d, localPath, key)
	default:
		return fmt.Errorf("unknown destination type %q", d.Type)
	}
}

// --- S3 (AWS Signature Version 4) -------------------------------------------

func uploadS3(ctx context.Context, d Destination, localPath, key string) error {
	data, err := os.ReadFile(localPath)
	if err != nil {
		return err
	}
	region := d.Region
	if region == "" {
		region = "us-east-1"
	}
	host := strings.TrimPrefix(strings.TrimPrefix(d.Endpoint, "https://"), "http://")
	host = strings.TrimSuffix(host, "/")
	if host == "" {
		host = fmt.Sprintf("s3.%s.amazonaws.com", region)
	}
	objectPath := "/" + d.Bucket + "/" + strings.TrimPrefix(key, "/")
	endpoint := "https://" + host + objectPath

	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")

	payloadHash := sha256Hex(data)
	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n", host, payloadHash, amzDate)
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"

	canonicalRequest := strings.Join([]string{
		http.MethodPut, objectPath, "",
		canonicalHeaders, signedHeaders, payloadHash,
	}, "\n")

	scope := strings.Join([]string{dateStamp, region, "s3", "aws4_request"}, "/")
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256", amzDate, scope, sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	signingKey := deriveSigningKey(d.SecretKey, dateStamp, region, "s3")
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	authorization := fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		d.AccessKey, scope, signedHeaders, signature,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	req.Header.Set("Authorization", authorization)
	req.ContentLength = int64(len(data))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("s3 upload failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func deriveSigningKey(secret, dateStamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	return hmacSHA256(kService, []byte("aws4_request"))
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// --- WebDAV -----------------------------------------------------------------

func uploadWebDAV(ctx context.Context, d Destination, localPath, key string) error {
	f, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}

	base := strings.TrimSuffix(d.Endpoint, "/")
	remote := base
	if d.Bucket != "" {
		remote += "/" + strings.Trim(d.Bucket, "/")
	}
	remote += "/" + strings.TrimPrefix(key, "/")

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, remote, f)
	if err != nil {
		return err
	}
	req.ContentLength = info.Size()
	if d.AccessKey != "" {
		req.SetBasicAuth(d.AccessKey, d.SecretKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("webdav upload failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

// --- SFTP/SSH ---------------------------------------------------------------

// uploadSFTP streams the file to the remote host over SSH by writing it to a
// remote `cat` into the target path. Bucket is treated as the remote directory.
func uploadSFTP(ctx context.Context, d Destination, localPath, key string) error {
	host := d.Endpoint
	if !strings.Contains(host, ":") {
		host += ":22"
	}
	cfg := &ssh.ClientConfig{
		User:            d.AccessKey,
		Auth:            []ssh.AuthMethod{ssh.Password(d.SecretKey)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}
	client, err := ssh.Dial("tcp", host, cfg)
	if err != nil {
		return fmt.Errorf("ssh dial: %w", err)
	}
	defer client.Close()

	remoteDir := strings.TrimSuffix(d.Bucket, "/")
	if remoteDir == "" {
		remoteDir = "."
	}
	remotePath := path.Join(remoteDir, key)

	f, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer f.Close()

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}
	// Single-quote the path safely so the remote shell treats it literally.
	quoted := "'" + strings.ReplaceAll(remotePath, "'", `'\''`) + "'"
	if err := session.Start("mkdir -p " + "'" + strings.ReplaceAll(remoteDir, "'", `'\''`) + "'" + " && cat > " + quoted); err != nil {
		return err
	}
	copyErr := make(chan error, 1)
	go func() {
		_, err := io.Copy(stdin, f)
		stdin.Close()
		copyErr <- err
	}()

	done := make(chan error, 1)
	go func() { done <- session.Wait() }()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		if err != nil {
			return fmt.Errorf("remote write: %w", err)
		}
	}
	if err := <-copyErr; err != nil {
		return err
	}
	return nil
}
