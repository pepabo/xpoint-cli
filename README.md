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

### クエリ一覧 / 実行

```sh
xp query list                           # 利用可能なクエリを一覧表示
xp query list --jq '.query_groups[].queries[].query_code'

xp query exec query01                   # クエリを実行して定義と結果を取得
xp query exec query01 --no-run          # 定義のみ取得（実行しない）
xp query exec query01 --rows 100 --offset 0
xp query exec query01 --jq '.exec_result.data'
```

### ドキュメント検索

```sh
xp document search --body '{"title":"経費"}'
xp document search --body ./search.json
cat search.json | xp document search --body -
xp document search --size 100 --page 2
```

フィルタフラグで簡易検索もできます（`--body` とは併用不可）。

```sh
xp document search --title 経費                       # 件名部分一致
xp document search --form-name 稟議 --form-group-id 3 # フォーム名 + フォームグループID
xp document search --writer alice --writer bob        # 申請者指定（複数可）
xp document search --writer-group grp1                # 申請者グループ指定
xp document search --me                               # 自分が申請者の書類（XPOINT_USER、未設定なら /scim/v2/{domain_code}/Me の atled 拡張 userCode を利用。domain_code は保存済み OAuth トークンの値も利用）
xp document search --since 2024-01-01 --until 2024-12-31
```

### ドキュメントの承認状況取得

```sh
xp document status 266248                   # 最新版の承認状況（JSON）
xp document status 266248 --history         # 全バージョンの承認履歴も含める
xp document status 266248 --jq '.document.status.name'
```

### ドキュメントをブラウザで開く

```sh
xp document open 266248              # 既定のブラウザで書類を開く
xp document open 266248 --no-browser # URLだけ出力（ブラウザは起動しない）
```

### ドキュメントのコメント操作

```sh
xp document comment get 266248                              # コメント一覧
xp document comment add 266248 --content "承認お願いします"  # コメント追加
xp document comment add 266248 --content "重要" --attention  # 重要コメント
xp document comment edit 266248 2 --content "修正後"         # 内容を更新
xp document comment edit 266248 2 --attention 1              # 重要フラグだけ更新
xp document comment delete 266248 2                          # コメント削除（確認あり）
xp document comment delete 266248 2 -y                       # 確認なしで削除
```

### ドキュメントのPDFダウンロード

```sh
xp document download 266248                # カレントディレクトリにサーバ提供ファイル名で保存
xp document download 266248 -o out.pdf     # 指定パスに保存
xp document download 266248 -o pdfs/       # 指定ディレクトリにサーバ提供ファイル名で保存
xp document download 266248 -o - > out.pdf # 標準出力に書き出し
```

### 認証ユーザー情報の確認

```sh
xp me                  # GET /scim/v2/{domain_code}/Me の結果を表示
xp me --jq .userName
```

OAuth 認証済みであることが前提です（汎用APIトークンでは SCIM は使えません）。`domain_code` は `--xpoint-domain-code` / `XPOINT_DOMAIN_CODE` / 保存済み OAuth トークンの順で解決されます。

`user_code` は X-point の内部ユーザコード（例: `326`）で、`document search --writer` などの writer_list API で使う値です。`user_name` は SCIM の `userName`（ログイン名、例: `ykky`）です。

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
