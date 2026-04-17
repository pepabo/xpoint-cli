# xpoint-cli

[X-point](https://atled-workflow.github.io/X-point-doc/api/) の REST API を使うためのコマンドラインクライアント `xp` です。

## 特徴

- OAuth2 Authorization Code + PKCE でのログイン（アクセストークンはシステムのキーリングに保存）
- 汎用APIトークン（`X-ATLED-Generic-API-Token`）やアクセストークンの直接指定にも対応
- フォーム一覧、承認一覧、ドキュメント検索などの主要APIに対応
- 出力は TTY 時はテーブル、リダイレクト時は JSON の自動切り替え
- `--jq` による gojq 形式のフィルタ
- `xp schema` で各操作のレスポンススキーマを確認可能

## インストール

Go 1.26 以降:

```sh
go install github.com/pepabo/xpoint-cli@latest
```

または [GitHub Releases](https://github.com/pepabo/xpoint-cli/releases) からバイナリを取得。

## 認証

以下の優先順で認証情報が解決されます。

1. コマンドラインフラグ（`--xpoint-*`）
2. 環境変数（`XPOINT_*`）
3. `xp auth login` で保存されたキーリング上の OAuth トークン

### OAuth2 でのログイン（推奨）

```sh
xp auth login \
  --xpoint-subdomain your-subdomain \
  --xpoint-domain-code your-domain \
  --xpoint-client-id your-client-id
```

環境変数 `XPOINT_SUBDOMAIN` / `XPOINT_DOMAIN_CODE` / `XPOINT_CLIENT_ID` でも指定できます。

ブラウザが起動して X-point の認可画面が開き、`http://127.0.0.1:<random-port>/callback` にリダイレクトされると認証が完了します。X-point 側のクライアント設定で `http://127.0.0.1` の任意ポート・パス `/callback` を許可しておく必要があります。

ブラウザを起動したくない場合は `--no-browser` を付けると URL が表示されます。

一度ログインすると、サブドメイン・アクセストークン・リフレッシュトークンはシステムキーリング（Linux: Secret Service / macOS: Keychain / Windows: Credential Manager）に単一のエントリとして保存され、以降のコマンドで自動利用されます。`XPOINT_SUBDOMAIN` も保存済みの値が使われるため省略可能です。複数アカウントの同時保存には対応しておらず、`xp auth login` は常に既存のエントリを上書きします。

保存済みトークンを確認:

```sh
xp auth status
```

### 汎用APIトークンを使う

```sh
export XPOINT_SUBDOMAIN=your-subdomain
export XPOINT_DOMAIN_CODE=your-domain
export XPOINT_USER=your-user
export XPOINT_GENERIC_API_TOKEN=xxxx
```

### 既存のアクセストークンを直接使う

```sh
export XPOINT_SUBDOMAIN=your-subdomain
export XPOINT_API_ACCESS_TOKEN=xxxx
```

## 使い方

### フォーム一覧

```sh
xp form list
xp form list -o json
xp form list --jq '.form_group[].form[].name'
```

### 承認一覧

```sh
xp approval list --stat 10          # 承認待ち
xp approval list --stat 50 --fgid 1 # 承認完了のうち form group 1
xp approval list --filter 'cr_dt between "2023-01-01" and "2023-12-31"'
```

`--stat` の値は X-point のマニュアル参照（10=承認待ち、20=通知、30=下書き等、40=状況確認、50=承認完了）。

### ドキュメント検索

```sh
xp document search --body '{"title":"経費"}'
xp document search --body ./search.json
cat search.json | xp document search --body -
xp document search --size 100 --page 2
```

### ドキュメントのPDFダウンロード

```sh
xp document download 266248                # カレントディレクトリにサーバ提供ファイル名で保存
xp document download 266248 -o out.pdf     # 指定パスに保存
xp document download 266248 -o pdfs/       # 指定ディレクトリにサーバ提供ファイル名で保存
xp document download 266248 -o - > out.pdf # 標準出力に書き出し
```

### レスポンススキーマの確認

```sh
xp schema              # エイリアス一覧
xp schema form.list
xp schema approval.list
xp schema document.search
```

## 出力フォーマット

- `-o table`: TTY 既定。整形済みテーブル
- `-o json`: パイプ／リダイレクト時の既定。生の JSON
- `--jq <expr>`: gojq フィルタを適用（JSON 出力に切り替わる）

## 環境変数

| 変数 | 説明 |
| --- | --- |
| `XPOINT_SUBDOMAIN` | サブドメイン（`https://<sub>.atledcloud.jp/xpoint`） |
| `XPOINT_DOMAIN_CODE` | ドメインコード（汎用APIトークンおよび OAuth ログイン時に必要） |
| `XPOINT_USER` | ユーザーコード（汎用APIトークン時） |
| `XPOINT_GENERIC_API_TOKEN` | 汎用APIトークン |
| `XPOINT_API_ACCESS_TOKEN` | OAuth2 アクセストークン |
| `XPOINT_CLIENT_ID` | OAuth2 クライアントID |
| `XP_DEBUG` | 非空で HTTP リクエスト/レスポンスのメタ情報を stderr に出力（エラー時はボディも出力） |

## ライセンス

[MIT License](./LICENSE)
