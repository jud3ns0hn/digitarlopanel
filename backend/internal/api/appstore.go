package api

// storeApp is a curated, one-click docker-compose application.
type storeApp struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Port        int    `json:"port"` // primary host port, for the "open" link
	Compose     string `json:"-"`    // docker-compose template
}

// appCatalog is the curated marketplace, grouped by Category.
var appCatalog = []storeApp{
	// ---- AI & Agents ----
	{
		Key: "ollama", Name: "Ollama", Category: "KI & Agenten", Port: 11434,
		Description: "Self-hosted LLM-Runtime zum Ausführen von Modellen wie Llama, Hermes, Mistral.",
		Compose: `services:
  ollama:
    image: ollama/ollama:latest
    restart: unless-stopped
    ports:
      - "11434:11434"
    volumes:
      - ollama:/root/.ollama
volumes:
  ollama:
`,
	},
	{
		Key: "open-webui", Name: "Open WebUI", Category: "KI & Agenten", Port: 3000,
		Description: "Komfortable Chat-Oberfläche für Ollama und OpenAI-kompatible Modelle.",
		Compose: `services:
  open-webui:
    image: ghcr.io/open-webui/open-webui:main
    restart: unless-stopped
    ports:
      - "3000:8080"
    environment:
      - OLLAMA_BASE_URL=http://host.docker.internal:11434
    extra_hosts:
      - "host.docker.internal:host-gateway"
    volumes:
      - open-webui:/app/backend/data
volumes:
  open-webui:
`,
	},
	{
		Key: "flowise", Name: "Flowise", Category: "KI & Agenten", Port: 3001,
		Description: "Low-Code-Builder für LLM-Agenten und -Workflows (drag & drop).",
		Compose: `services:
  flowise:
    image: flowiseai/flowise:latest
    restart: unless-stopped
    ports:
      - "3001:3000"
    volumes:
      - flowise:/root/.flowise
volumes:
  flowise:
`,
	},
	{
		Key: "anythingllm", Name: "AnythingLLM", Category: "KI & Agenten", Port: 3002,
		Description: "All-in-one RAG/Chat-App für eigene Dokumente und Agenten.",
		Compose: `services:
  anythingllm:
    image: mintplexlabs/anythingllm:latest
    restart: unless-stopped
    ports:
      - "3002:3001"
    volumes:
      - anythingllm:/app/server/storage
volumes:
  anythingllm:
`,
	},
	{
		Key: "librechat", Name: "LibreChat", Category: "KI & Agenten", Port: 3080,
		Description: "Open-Source ChatGPT-Alternative mit vielen Provider- und Agenten-Funktionen.",
		Compose: `services:
  librechat:
    image: ghcr.io/danny-avila/librechat:latest
    restart: unless-stopped
    ports:
      - "3080:3080"
    environment:
      - HOST=0.0.0.0
      - MONGO_URI=mongodb://mongo:27017/LibreChat
    depends_on:
      - mongo
  mongo:
    image: mongo:7
    restart: unless-stopped
    volumes:
      - librechat-mongo:/data/db
volumes:
  librechat-mongo:
`,
	},
	// ---- Automation ----
	{
		Key: "n8n", Name: "n8n", Category: "Automatisierung", Port: 5678,
		Description: "Mächtige Workflow-Automatisierung mit Hunderten Integrationen und KI-Knoten.",
		Compose: `services:
  n8n:
    image: docker.n8n.io/n8nio/n8n:latest
    restart: unless-stopped
    ports:
      - "5678:5678"
    environment:
      - N8N_SECURE_COOKIE=false
    volumes:
      - n8n:/home/node/.n8n
volumes:
  n8n:
`,
	},
	{
		Key: "node-red", Name: "Node-RED", Category: "Automatisierung", Port: 1880,
		Description: "Flow-basierte Programmierung für IoT, Automation und APIs.",
		Compose: `services:
  node-red:
    image: nodered/node-red:latest
    restart: unless-stopped
    ports:
      - "1880:1880"
    volumes:
      - node-red:/data
volumes:
  node-red:
`,
	},
	// ---- Databases ----
	{
		Key: "postgres", Name: "PostgreSQL", Category: "Datenbanken", Port: 5432,
		Description: "Robuste relationale Datenbank.",
		Compose: `services:
  postgres:
    image: postgres:16
    restart: unless-stopped
    ports:
      - "5432:5432"
    environment:
      - POSTGRES_PASSWORD=changeme
    volumes:
      - postgres:/var/lib/postgresql/data
volumes:
  postgres:
`,
	},
	{
		Key: "redis", Name: "Redis", Category: "Datenbanken", Port: 6379,
		Description: "In-Memory Key-Value-Store / Cache.",
		Compose: `services:
  redis:
    image: redis:7
    restart: unless-stopped
    ports:
      - "6379:6379"
    volumes:
      - redis:/data
volumes:
  redis:
`,
	},
	{
		Key: "mongodb", Name: "MongoDB", Category: "Datenbanken", Port: 27017,
		Description: "Dokumentenorientierte NoSQL-Datenbank.",
		Compose: `services:
  mongodb:
    image: mongo:7
    restart: unless-stopped
    ports:
      - "27017:27017"
    volumes:
      - mongodb:/data/db
volumes:
  mongodb:
`,
	},
	// ---- Monitoring ----
	{
		Key: "uptime-kuma", Name: "Uptime Kuma", Category: "Monitoring", Port: 3003,
		Description: "Schönes Self-Hosted Uptime-Monitoring mit Benachrichtigungen.",
		Compose: `services:
  uptime-kuma:
    image: louislam/uptime-kuma:1
    restart: unless-stopped
    ports:
      - "3003:3001"
    volumes:
      - uptime-kuma:/app/data
volumes:
  uptime-kuma:
`,
	},
	{
		Key: "grafana", Name: "Grafana", Category: "Monitoring", Port: 3004,
		Description: "Dashboards und Visualisierung für Metriken und Logs.",
		Compose: `services:
  grafana:
    image: grafana/grafana:latest
    restart: unless-stopped
    ports:
      - "3004:3000"
    volumes:
      - grafana:/var/lib/grafana
volumes:
  grafana:
`,
	},
	// ---- Productivity ----
	{
		Key: "nextcloud", Name: "Nextcloud", Category: "Produktivität", Port: 8080,
		Description: "Eigene Cloud für Dateien, Kalender und Kontakte.",
		Compose: `services:
  nextcloud:
    image: nextcloud:latest
    restart: unless-stopped
    ports:
      - "8081:80"
    volumes:
      - nextcloud:/var/www/html
volumes:
  nextcloud:
`,
	},
	{
		Key: "vaultwarden", Name: "Vaultwarden", Category: "Produktivität", Port: 8082,
		Description: "Self-hosted Passwortmanager (Bitwarden-kompatibel).",
		Compose: `services:
  vaultwarden:
    image: vaultwarden/server:latest
    restart: unless-stopped
    ports:
      - "8082:80"
    volumes:
      - vaultwarden:/data
volumes:
  vaultwarden:
`,
	},
	{
		Key: "gitea", Name: "Gitea", Category: "Produktivität", Port: 3005,
		Description: "Leichtgewichtiger, self-hosted Git-Service.",
		Compose: `services:
  gitea:
    image: gitea/gitea:latest
    restart: unless-stopped
    ports:
      - "3005:3000"
      - "2222:22"
    volumes:
      - gitea:/data
volumes:
  gitea:
`,
	},
}

func findStoreApp(key string) (storeApp, bool) {
	for _, a := range appCatalog {
		if a.Key == key {
			return a, true
		}
	}
	return storeApp{}, false
}
