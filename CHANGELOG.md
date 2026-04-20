# Changelog

## [v1.1.1](https://github.com/pepabo/xpoint-cli/compare/v1.1.0...v1.1.1) - 2026-04-20
- release: リリース成果物に winget manifest を追加 by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/39

## [v1.1.0](https://github.com/pepabo/xpoint-cli/compare/v1.0.0...v1.1.0) - 2026-04-17
- goreleaser: アーカイブ名をxp_{version}_{os}_{arch}に変更 by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/16
- document: PDFダウンロードサブコマンドを追加 by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/18
- document: 承認状況取得サブコマンドを追加 by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/20
- document: ブラウザで書類を開くopenサブコマンドを追加 by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/21
- query: 一覧取得・実行サブコマンドを追加 by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/22
- document: コメントの追加・取得・更新・削除サブコマンドを追加 by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/23
- document: search にフィルタフラグを追加 by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/19
- document: 添付ファイルの追加・取得・一覧・更新・削除サブコマンドを追加 by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/24
- approval: 承認待ち件数取得・承認完了書類非表示設定サブコマンドを追加 by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/26
- document: 書類ビュー (docview/openview/statusview) 取得サブコマンドを追加 by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/27
- query: クエリグラフ取得サブコマンドを追加 by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/28
- system: 登録フォーム一覧・フォーム定義取得サブコマンドを追加 by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/29
- system: マスタ管理サブコマンド (list/show/data/import/upload) を追加 by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/33
- table: ヘッダ下線付きのテーブル/リスト出力に刷新 by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/32
- system: Webhookログ取得・詳細取得サブコマンドを追加 by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/34
- system: Webhook設定の取得・登録・更新・削除サブコマンドを追加 by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/35
- misc: adminrole / proxy / service 取得サブコマンドを追加 by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/36
- system: 自動申請 (lumpapply) 一覧・定義取得サブコマンドを追加 by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/31
- misc: .gitignoreのClaude Code worktreeパスを更新 by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/38
- document: status を TTY 向けにテーブル/リスト表示へ刷新 by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/37

## [v1.0.0](https://github.com/pepabo/xpoint-cli/commits/v1.0.0) - 2026-04-17
- Add tests, CI workflow, and golangci-lint config by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/1
- Add goreleaser and tagpr for automated releases by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/2
- Use GitHub App token for tagpr by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/3
- Add xp schema command by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/4
- OAuth2 Authorization Code + PKCE 認証を追加 by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/6
- tagpr: versionFileをcmd/root.goに設定 by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/7
- docs: READMEを拡充 by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/8
- Add xp document create command by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/9
- version: 1.0.0 に更新しcommit/dateを削除 by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/10
- document: get/edit/delete サブコマンドを追加 by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/11
- Add xp form show and document URL output by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/12
- auth: keyringの保存をsubdomain非依存の単一エントリに変更 by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/13
- Release for v1.0.0 by @pepabo-pr-maker[bot] in https://github.com/pepabo/xpoint-cli/pull/5
- tagpr: release自動化を無効化しapp-id deprecation対応 by @buty4649 in https://github.com/pepabo/xpoint-cli/pull/15
