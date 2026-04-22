<div align="center">
  <img src="https://go.dev/blog/go-brand/Go-Logo/PNG/Go-Logo_Blue.png" alt="Go Logo" width="220" />

  # AliExpress Tracker Bot

  Find products on AliExpress and receive alerts on Telegram.
</div>

## What this bot does
This bot searches AliExpress using the keywords you choose.
Then it sends a Telegram message with:
- product name
- current price
- original price (when available)
- discount percentage
- rating, sales, and link

If you want, it can show **only deals** with a minimum discount (for example: 10%).

## Telegram output example

```text
AliExpress search summary:

Search: xeon x99 kit
1. Xeon X99 Kit...
   Price: $170.00 (before $200.00) | Discount: 15.0% | Rating: 4.8 | Sales: 1200
   Link: https://www.aliexpress.com/item/...
```

## Step 1: create your Telegram bot
1. Open Telegram and search for `@BotFather`.
2. Send `/newbot`.
3. Choose a name and username.
4. Copy the token BotFather gives you (this is `TELEGRAM_TOKEN`).

## Step 2: get your CHAT ID
1. Start a chat with your bot and send any message (for example: `hi`).
2. Open this URL in your browser:

```text
https://api.telegram.org/botYOUR_TOKEN/getUpdates
```

3. In the JSON response, find `"chat"` and copy `"id"`.
4. This number is your `TELEGRAM_CHAT_ID`.

## Step 3: set environment variables (.env)
Create a `.env` file from `.env.example`.

PowerShell (Windows):

```powershell
Copy-Item .env.example .env
```

Then fill in real values:

```env
RAPIDAPI_KEY=your_rapidapi_key
RAPIDAPI_HOST=aliexpress-datahub.p.rapidapi.com
TELEGRAM_TOKEN=your_token
TELEGRAM_CHAT_ID=your_chat_id
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

### Important deal filter variable
- `TRACKER_MIN_DISCOUNT_PERCENT=10`
  - Shows only products with 10% discount or more.
- Use `0` to show everything.

## Run locally
From the project folder:

```bash
go mod tidy
go run ./src
```

## Run automatically with GitHub Actions (recommended)
This project already includes:
- `.github/workflows/tracker.yml`

### 1) Push project to GitHub
- Push your code to your repository.

### 2) Configure Secrets
Go to:
- `Settings > Secrets and variables > Actions > New repository secret`

Create:
- `RAPIDAPI_KEY`
- `TELEGRAM_TOKEN`
- `TELEGRAM_CHAT_ID`

### 3) Run once manually
- Open `Actions`
- Select `Ali Tracker`
- Click `Run workflow`

### 4) Schedule (cron)
It is already configured to run 3 times a day.
If needed, change `cron` in `tracker.yml`.

## How the discount filter works
The bot calculates:

```text
discount % = (original_price - promo_price) / original_price * 100
```

It only sends products with discount >= `TRACKER_MIN_DISCOUNT_PERCENT`.

## Common issues
### "missing required env variable"
A required variable is missing in `.env`.

### Bot does not send messages
Check:
- `TELEGRAM_TOKEN`
- `TELEGRAM_CHAT_ID`
- whether you sent at least one message to the bot

### No products found
- Try different keywords in `ALI_SEARCH_TERMS`.
- Lower `TRACKER_MIN_DISCOUNT_PERCENT` to `0` or `5`.

## Project structure
- `src/main.go`: app entry and env loading
- `src/aliexpress/client.go`: AliExpress API calls
- `src/tracker/worker.go`: filtering and message assembly
- `src/telegram/bot.go`: Telegram sender
- `src/domain/product.go`: product model

## License
This project is licensed under the MIT License.
See [LICENSE](LICENSE) for details.
