<div align="center">
  <img src="https://go.dev/blog/go-brand/Go-Logo/PNG/Go-Logo_Blue.png" alt="Go Logo" width="220" />

  # AliExpress Tracker Bot

  Encontre produtos no AliExpress e receba alertas no Telegram.
</div>

## O que este bot faz
Este bot pesquisa produtos no AliExpress pelos termos que voce definir.
Depois ele envia no Telegram:
- nome do produto
- preco atual
- preco original (quando existir)
- percentual de desconto
- nota, vendas e link

Se voce quiser, ele pode mostrar **so promocoes** com desconto minimo (ex.: 10%).

## Resultado final no Telegram
Voce recebe uma mensagem tipo:

```text
Resumo de buscas AliExpress :

Busca: kit xeon x99
1. Kit Xeon X99...
   Preco: R$ 850,00 (antes R$ 1.000,00) | Desconto: 15.0% | Nota: 4.8 | Vendas: 1200
   Link: https://www.aliexpress.com/item/...
```

## Passo 1: criar seu bot no Telegram
1. Abra o Telegram e procure por `@BotFather`.
2. Envie `/newbot`.
3. Escolha um nome e um username para o bot.
4. Copie o token que o BotFather te entregar (esse sera o `TELEGRAM_TOKEN`).

## Passo 2: pegar seu CHAT ID
1. Inicie conversa com seu bot e envie qualquer mensagem (ex.: `oi`).
2. No navegador, abra:

```text
https://api.telegram.org/botSEU_TOKEN/getUpdates
```

3. Procure no retorno JSON por `"chat"` e pegue o valor de `"id"`.
4. Esse numero e o `TELEGRAM_CHAT_ID`.

## Passo 3: configurar variaveis (.env)
Crie um arquivo `.env` na raiz do projeto com base no `.env.example`.

PowerShell (Windows):

```powershell
Copy-Item .env.example .env
```

Depois preencha os valores reais:

```env
RAPIDAPI_KEY=coloque_sua_chave_rapidapi
RAPIDAPI_HOST=aliexpress-datahub.p.rapidapi.com
TELEGRAM_TOKEN=coloque_seu_token
TELEGRAM_CHAT_ID=coloque_seu_chat_id
ALI_REGION=BR
ALI_LOCALE=pt_BR
ALI_CURRENCY=BRL
ALI_SEARCH_TERMS=kit xeon x99,memoria ram,placa de video
TRACKER_TOP_N=5
TRACKER_MIN_DISCOUNT_PERCENT=10
TRACKER_ENABLE_SCHEDULER=false
TRACKER_RUN_ON_START=true
TRACKER_RUN_TIMES=09:00,15:00,21:00
TRACKER_TIMEZONE=America/Sao_Paulo
```

### Variavel importante para promocao
- `TRACKER_MIN_DISCOUNT_PERCENT=10`
  - So mostra produtos com 10% ou mais de desconto.
- Se quiser ver tudo, use `0`.

## Rodar localmente
No terminal, dentro da pasta do projeto:

```bash
go mod tidy
go run ./src
```

## Rodar automatico no GitHub Actions 
Este projeto ja tem workflow em:
- `.github/workflows/tracker.yml`

### 1) Suba o projeto no GitHub
- Faça push do codigo para seu repositorio.

### 2) Configure os Secrets
No GitHub, entre em:
- `Settings > Secrets and variables > Actions > New repository secret`

Crie:
- `RAPIDAPI_KEY`
- `TELEGRAM_TOKEN`
- `TELEGRAM_CHAT_ID`

### 3) Rode manualmente uma vez
- Aba `Actions`
- Workflow `Ali Tracker`
- Clique em `Run workflow`

### 4) Agendamento (cron)
Ja esta configurado para rodar 3x ao dia.
Se quiser mudar, edite o `cron` no arquivo `tracker.yml`.

## Como o filtro de promocao funciona
O bot calcula:

```text
desconto % = (preco_original - preco_promocional) / preco_original * 100
```

Ele so manda o produto se o desconto for maior ou igual ao valor de `TRACKER_MIN_DISCOUNT_PERCENT`.

## Problemas comuns
### "missing required env variable"
Alguma variavel obrigatoria no `.env` esta vazia.

### Bot nao envia mensagem
Confira:
- `TELEGRAM_TOKEN`
- `TELEGRAM_CHAT_ID`
- se voce enviou mensagem para o bot antes

### Nao apareceu nenhum produto
- Tente termos diferentes em `ALI_SEARCH_TERMS`.
- Diminua `TRACKER_MIN_DISCOUNT_PERCENT` para `0` ou `5`.

## Estrutura do projeto
- `src/main.go`: inicia o app e carrega variaveis
- `src/aliexpress/client.go`: consulta API do AliExpress
- `src/tracker/worker.go`: aplica filtro e monta mensagem
- `src/telegram/bot.go`: envia mensagem para Telegram
- `src/domain/product.go`: modelo de produto

## Licenca
Este projeto esta licenciado sob a licenca MIT.  
Veja o arquivo [LICENSE](LICENSE) para mais detalhes.