# Reorder Playlist TUI

Um aplicativo de terminal interativo (TUI) para reordenar playlists do YouTube e salvar a nova ordem no próprio YouTube. A aplicação utiliza OAuth2 para autenticação com Google, mantém logs em JSON e armazena token localmente.

## Funcionalidades

- Autenticação OAuth2 com Google / YouTube
- Listar playlists do usuário
- Exibir vídeos de cada playlist
- Reordenar playlist localmente por:

    * Nome
    * Duração
    * Idioma
    * Data de publicação
* Permitir ao usuário digitar um novo título para a playlist antes de salvar
* Salvar nova playlist (com nova ordem e novo título) no YouTube
* Exibir indicador de “loading” de 10 segundos durante o salvamento
* Arquivos de log em JSON (níveis INFO, WARNING e ERROR)
* Armazenar e atualizar token OAuth em disco (`token.json` por padrão)

## Pré-requisitos

* Go 1.20+ instalado na máquina
* Conta Google com playlists existentes
* Arquivo de credenciais OAuth2 (`client_secret.json`)
* Conexão com Internet (para comunicar-se com a API do YouTube)

## Como instalar

```bash
git clone https://github.com/SEU_USUARIO/TUI_playlist_reorder.git
cd TUI_playlist_reorder
go mod download
go build -o reorder-tui ./cmd
```

## Configuração

### Credenciais de OAuth2

1. Crie um projeto no Google Cloud Console.
2. Ative a API do YouTube Data.
3. Na seção “Credenciais”, crie um OAuth 2.0 Client ID (tipo “Desktop”).
4. Baixe o arquivo JSON com as credenciais (`client_secret.json`) e coloque-o na pasta auth.

### Configurar redirect URI

* O redirect URI padrão é `http://localhost:8080/`.

### Token local

* Na primeira execução, a TUI abrirá uma URL para login.
* Após a autorização, um `token.json` será criado na pasta `token_manager`
* Em execuções futuras, o token será recarregado automaticamente.

## Como usar

### 1. Executar a aplicação

```bash
go run main.go
```

### 2. Autenticar com Google

A TUI exibirá a tela:

```
🎶 Reorder Playlist TUI 🎶
Pressione Enter para login com Google e começar a reordenar!
(Press Ctrl+C ou Esc para sair)
```

Será gerado um link OAuth2. Copie e cole no navegador se não abrir automaticamente.

### 3. Selecionar e reordenar playlist

```
Your Playlists
-------------
> Minha Playlist 1
  Outra Playlist
  Playlist de Exemplo
```

Use ↑/↓ ou j/k para navegar. Enter para selecionar.

### 4. Digitar novo título e salvar

Menu de reordenação:

```
Reordenar Playlist: Minha Playlist 1

Vídeos desta playlist (ordem atual):
- Vídeo A
- Vídeo B
- Vídeo C

Opções de Reordenação:
> Ordenar por Nome (A-Z)
  Ordenar por Duração (Menor-Maior)
  Ordenar por Idioma (A-Z)
  Ordenar por Data de Publicação (Mais Antigo-Mais Novo)
  Voltar para Playlists
```

Digite o novo título:

```
Digite o novo título para a playlist e pressione Enter:
> Meu Novo Título
```

Loading:

```
⏳ Salvando playlist "Meu Novo Título" no YouTube. Aguarde...
```

## Estrutura do Projeto

```
TUI_playlist_reorder/
├── cmd/
│   └── main.go
├── internal/
│   ├── auth/
│   ├── core/
│   │   ├── domain/
│   │   └── usecases/
│   ├── handler/
│   │   ├── server/
│   │   └── tui/
│   ├── infrastructure/
│   │   ├── logger/
│   │   ├── token_manager/
│   │   └── provider/
│   └── ports/
├── configs/
│   └── client_secret.json
├── go.mod
├── go.sum
├── .gitignore
└── README.md
```

## Detalhes de Implementação

### Autenticação (auth)

* Gera URL de autenticação
* Recebe callback HTTP
* Recarrega token

### Provedor do YouTube (provider)

* Listar playlists e vídeos
* Criar nova playlist
* Adicionar vídeos à playlist
* Uso de `sync.Mutex` para inicialização do cliente

### Use Cases (usecases)

* `GetMinePlaylists`
* `ReorderPlaylist`

    * Ordenação local
    * Criação de nova playlist
    * Registro de logs

### TUI (tui)

* Baseado em Bubble Tea
* Modelos:

    * `WelcomeModel`
    * `LoginModel`
    * `PlaylistsModel`
    * `ReorderModel`
* Estilização com Lip Gloss

### Logger (logger)

* Interface com métodos Info, Error, Warning
* JSON logs com timestamp, arquivo, função

### Token Manager (token\_manager)

* Grava e carrega token OAuth2 em JSON local

### Callback HTTP (server)

* Recebe código OAuth2 via callback
* Verificação de CSRF
* Encerramento automático