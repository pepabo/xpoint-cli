package xpoint

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	auth    Auth
	http    *http.Client
}

// Auth holds X-point credentials. Exactly one of GenericAPIToken or AccessToken must be set.
type Auth struct {
	// GenericAPIToken auth: sent as X-ATLED-Generic-API-Token.
	DomainCode      string
	User            string
	GenericAPIToken string

	// OAuth2 access token: sent as Authorization: Bearer.
	AccessToken string
}

// redactToken returns a partial preview of a credential for debug output:
// first and last few chars only.
func redactToken(s string) string {
	if len(s) <= 8 {
		return "***"
	}
	return s[:4] + "***" + s[len(s)-4:]
}

func (a Auth) apply(req *http.Request) {
	if a.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+a.AccessToken)
		return
	}
	raw := fmt.Sprintf("%s:%s:%s", a.DomainCode, a.User, a.GenericAPIToken)
	req.Header.Set("X-ATLED-Generic-API-Token", base64.StdEncoding.EncodeToString([]byte(raw)))
}

func NewClient(subdomain string, auth Auth) *Client {
	return &Client{
		baseURL: fmt.Sprintf("https://%s.atledcloud.jp/xpoint", subdomain),
		auth:    auth,
		http:    &http.Client{Timeout: 120 * time.Second},
	}
}

type Route struct {
	ID   int    `json:"id"`
	Code string `json:"code,omitempty"`
	Name string `json:"name"`
}

type Form struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Code  string  `json:"code"`
	Route []Route `json:"route,omitempty"`
}

type FormGroup struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Form []Form `json:"form"`
}

type FormsListResponse struct {
	FormGroup []FormGroup `json:"form_group"`
}

// ListAvailableForms calls GET /api/v1/forms (ユーザー別API: 利用可能フォーム一覧取得).
func (c *Client) ListAvailableForms(ctx context.Context) (*FormsListResponse, error) {
	var out FormsListResponse
	if err := c.do(ctx, http.MethodGet, "/api/v1/forms", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type FormField struct {
	Seq       int    `json:"seq"`
	FieldID   string `json:"fieldid"`
	FieldType int    `json:"fieldtype"`
	MaxLength int    `json:"maxlength"`
	Label     string `json:"label"`
	GroupName string `json:"groupname"`
	ArraySize int    `json:"arraysize"`
	Required  bool   `json:"required"`
	Unique    bool   `json:"unique"`
}

type FormDetailPage struct {
	PageNo   int         `json:"page_no"`
	FormCode string      `json:"form_code,omitempty"`
	FormName string      `json:"form_name,omitempty"`
	Fields   []FormField `json:"fields"`
}

type FormDetailRoute struct {
	Code      string `json:"code,omitempty"`
	Name      string `json:"name"`
	Condroute bool   `json:"condroute"`
}

type FormDetail struct {
	Code    string            `json:"code"`
	Name    string            `json:"name"`
	MaxStep int               `json:"max_step"`
	Route   []FormDetailRoute `json:"route"`
	Pages   []FormDetailPage  `json:"pages"`
}

type FormDetailResponse struct {
	Form FormDetail `json:"form"`
}

// DocumentURL returns the web URL for a document, suitable for opening
// in a browser. The server redirects to the right form when only docid
// is provided.
func (c *Client) DocumentURL(docID int) string {
	if docID <= 0 {
		return ""
	}
	return fmt.Sprintf("%s/form.do?act=view&docid=%d", c.baseURL, docID)
}

// GetFormDetail calls GET /api/v1/forms/{fid} to fetch field definitions.
func (c *Client) GetFormDetail(ctx context.Context, formID int) (*FormDetailResponse, error) {
	path := fmt.Sprintf("/api/v1/forms/%d", formID)
	var out FormDetailResponse
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SystemForm is an entry in the admin-side form list (GET /api/v1/system/forms).
// It carries extra fields (page_count, table_name, tsffile_name) compared to
// the user-facing form list.
type SystemForm struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	PageCount   int    `json:"page_count"`
	TableName   string `json:"table_name"`
	TsfFileName string `json:"tsffile_name"`
}

type SystemFormGroup struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	FormCount int          `json:"form_count"`
	Form      []SystemForm `json:"form"`
}

type SystemFormsListResponse struct {
	FormGroup []SystemFormGroup `json:"form_group"`
}

// ListSystemForms calls GET /api/v1/system/forms (admin: 登録フォーム一覧取得).
func (c *Client) ListSystemForms(ctx context.Context) (*SystemFormsListResponse, error) {
	var out SystemFormsListResponse
	if err := c.do(ctx, http.MethodGet, "/api/v1/system/forms", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetSystemFormDetail calls GET /api/v1/system/forms/{fid} (admin) and returns
// the same shape as GetFormDetail.
func (c *Client) GetSystemFormDetail(ctx context.Context, formID int) (*FormDetailResponse, error) {
	path := fmt.Sprintf("/api/v1/system/forms/%d", formID)
	var out FormDetailResponse
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type Master struct {
	Type      int    `json:"type"`
	TypeName  string `json:"type_name"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	TableName string `json:"table_name"`
	ItemCount int    `json:"item_count"`
	Remarks   string `json:"remarks"`
}

type MasterListResponse struct {
	Master []Master `json:"master"`
}

// ListMasters calls GET /api/v1/system/master (admin).
func (c *Client) ListMasters(ctx context.Context) (*MasterListResponse, error) {
	var out MasterListResponse
	if err := c.do(ctx, http.MethodGet, "/api/v1/system/master", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type UserMasterField struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Length     any    `json:"length"` // varchar -> int, numeric -> "10.2"
	PrimaryKey bool   `json:"primary_key"`
	Index      bool   `json:"index"`
}

type UserMasterInfoResponse struct {
	TableName string            `json:"table_name"`
	Fields    []UserMasterField `json:"fields"`
}

// GetUserMasterInfo calls GET /api/v1/system/master/{master_table_name} (admin)
// and returns the user-specific master property definition.
func (c *Client) GetUserMasterInfo(ctx context.Context, tableName string) (*UserMasterInfoResponse, error) {
	path := fmt.Sprintf("/api/v1/system/master/%s", url.PathEscape(tableName))
	var out UserMasterInfoResponse
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// MasterDataParams holds query parameters for GET /api/v1/system/master/{master_code}/data.
// MasterType is required (0: simple master, 1: user-specific master).
// When the path suffix is .csv, the remaining fields (FileName, Delimiter,
// Title, Fields) apply.
type MasterDataParams struct {
	MasterType int    // required (0 or 1)
	Rows       *int   // default 100, max 1000
	Offset     *int   // default 0
	FileName   string // CSV only
	Delimiter  string // CSV only: "comma" | "tab"
	Title      *bool  // CSV + user-specific master only
	Fields     string // CSV + simple master only: comma-separated list
}

func (p MasterDataParams) query() url.Values {
	v := url.Values{}
	v.Set("master_type", strconv.Itoa(p.MasterType))
	if p.Rows != nil {
		v.Set("rows", strconv.Itoa(*p.Rows))
	}
	if p.Offset != nil {
		v.Set("offset", strconv.Itoa(*p.Offset))
	}
	if p.FileName != "" {
		v.Set("file_name", p.FileName)
	}
	if p.Delimiter != "" {
		v.Set("delimiter", p.Delimiter)
	}
	if p.Title != nil {
		v.Set("title", strconv.FormatBool(*p.Title))
	}
	if p.Fields != "" {
		v.Set("fields", p.Fields)
	}
	return v
}

// GetMasterData calls GET /api/v1/system/master/{master_code}/data[.json|.csv]
// and returns the server-provided filename, raw response bytes, and content
// type. suffix may be "", "json", or "csv" (mapped to .json/.csv).
func (c *Client) GetMasterData(ctx context.Context, masterCode, suffix string, p MasterDataParams) (string, []byte, string, error) {
	path := fmt.Sprintf("/api/v1/system/master/%s/data", url.PathEscape(masterCode))
	accept := "application/json"
	switch strings.ToLower(suffix) {
	case "", "json":
		// default: JSON
	case "csv":
		path += ".csv"
		accept = "text/csv"
	default:
		return "", nil, "", fmt.Errorf("unknown master data format %q (must be json or csv)", suffix)
	}
	filename, body, ct, err := c.downloadBytesWithContentType(ctx, http.MethodGet, path, p.query(), nil, "", accept)
	return filename, body, ct, err
}

type SimpleMasterDataItem struct {
	Code  string `json:"code"`
	Value any    `json:"value"`
}

type ImportSimpleMasterRequest struct {
	Overwrite *bool                  `json:"overwrite,omitempty"`
	Data      []SimpleMasterDataItem `json:"data"`
}

// ImportSimpleMasterData calls PUT /api/v1/system/master/{master_code}/data
// (admin) to import rows into a simple master. The response shape is
// documented but complex, so it is returned as raw JSON.
func (c *Client) ImportSimpleMasterData(ctx context.Context, masterCode string, req ImportSimpleMasterRequest) (json.RawMessage, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	path := fmt.Sprintf("/api/v1/system/master/%s/data", url.PathEscape(masterCode))
	var out json.RawMessage
	if err := c.do(ctx, http.MethodPut, path, nil, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type UploadUserMasterResponse struct {
	Master struct {
		Type      int    `json:"type"`
		TypeName  string `json:"type_name"`
		Name      string `json:"name"`
		Code      string `json:"code"`
		TableName string `json:"table_name"`
	} `json:"master"`
	MessageType int    `json:"message_type"`
	Message     string `json:"message"`
}

// UploadUserMasterCSV calls POST /multiapi/v1/system/master/{master_table_name}/data
// (admin) to upload a CSV file to a user-specific master's import staging.
// fileName is the CSV filename, content the raw CSV bytes, overwrite is sent
// as the overwrite form field when non-nil.
func (c *Client) UploadUserMasterCSV(ctx context.Context, tableName, fileName string, content []byte, overwrite *bool) (*UploadUserMasterResponse, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, fileName))
	h.Set("Content-Type", "text/csv")
	fw, err := mw.CreatePart(h)
	if err != nil {
		return nil, fmt.Errorf("create file part: %w", err)
	}
	if _, err := fw.Write(content); err != nil {
		return nil, fmt.Errorf("write file content: %w", err)
	}
	if overwrite != nil {
		if err := mw.WriteField("overwrite", strconv.FormatBool(*overwrite)); err != nil {
			return nil, fmt.Errorf("write overwrite: %w", err)
		}
	}
	if err := mw.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	path := fmt.Sprintf("/multiapi/v1/system/master/%s/data", url.PathEscape(tableName))
	_, body, _, err := c.downloadBytesWithContentType(ctx, http.MethodPost, path, nil, buf.Bytes(), mw.FormDataContentType(), "application/json")
	if err != nil {
		return nil, err
	}
	var out UploadUserMasterResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &out, nil
}

type WebhooklogEntry struct {
	DomainCode string `json:"domain_code"`
	DocID      string `json:"docid"`
	FormCode   string `json:"form_code"`
	RouteCode  string `json:"route_code"`
	Title1     string `json:"title1"`
	URL        string `json:"url"`
	StatusCode string `json:"status_code"`
	SendDate   string `json:"send_date"`
	UUID       string `json:"uuid"`
}

type WebhooklogListResponse struct {
	Data []WebhooklogEntry `json:"data"`
}

// WebhooklogListParams holds query parameters for GET /api/v1/system/webhooklog.
type WebhooklogListParams struct {
	From      string // from
	To        string // to
	DocID     *int   // docid
	FormCode  string // form_code
	RouteCode string // route_code
	Status    string // status: all | success | failed
	URL       string // url
	Limit     *int   // limit
	Offset    *int   // offset
}

func (p WebhooklogListParams) query() url.Values {
	v := url.Values{}
	if p.From != "" {
		v.Set("from", p.From)
	}
	if p.To != "" {
		v.Set("to", p.To)
	}
	if p.DocID != nil {
		v.Set("docid", strconv.Itoa(*p.DocID))
	}
	if p.FormCode != "" {
		v.Set("form_code", p.FormCode)
	}
	if p.RouteCode != "" {
		v.Set("route_code", p.RouteCode)
	}
	if p.Status != "" {
		v.Set("status", p.Status)
	}
	if p.URL != "" {
		v.Set("url", p.URL)
	}
	if p.Limit != nil {
		v.Set("limit", strconv.Itoa(*p.Limit))
	}
	if p.Offset != nil {
		v.Set("offset", strconv.Itoa(*p.Offset))
	}
	return v
}

// ListWebhooklog calls GET /api/v1/system/webhooklog (admin).
func (c *Client) ListWebhooklog(ctx context.Context, p WebhooklogListParams) (*WebhooklogListResponse, error) {
	var out WebhooklogListResponse
	if err := c.do(ctx, http.MethodGet, "/api/v1/system/webhooklog", p.query(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetWebhooklog calls GET /api/v1/system/webhooklog/{uuid} (admin). The
// response contains request/response detail whose body may be an arbitrary
// type, so it is returned as raw JSON.
func (c *Client) GetWebhooklog(ctx context.Context, uuid string) (json.RawMessage, error) {
	path := fmt.Sprintf("/api/v1/system/webhooklog/%s", url.PathEscape(uuid))
	var out json.RawMessage
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type WebhookConfig struct {
	ID      int    `json:"id"`
	URL     string `json:"url"`
	Remarks string `json:"remarks,omitempty"`
}

type WebhookListResponse struct {
	FormName string          `json:"form_name"`
	FormType string          `json:"form_type"`
	Webhooks []WebhookConfig `json:"webhooks"`
}

// ListWebhookConfigs calls GET /api/v1/system/{fid}/webhooks (admin). The
// fqdn query parameter is required; only configs whose destination URL FQDN
// exactly matches fqdn are returned.
func (c *Client) ListWebhookConfigs(ctx context.Context, formID int, fqdn string) (*WebhookListResponse, error) {
	path := fmt.Sprintf("/api/v1/system/%d/webhooks", formID)
	q := url.Values{}
	q.Set("fqdn", fqdn)
	var out WebhookListResponse
	if err := c.do(ctx, http.MethodGet, path, q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type CreateWebhookRequest struct {
	URL     string `json:"url"`
	Remarks string `json:"remarks,omitempty"`
}

// CreateWebhookConfig calls POST /api/v1/system/{fid}/webhooks (admin).
func (c *Client) CreateWebhookConfig(ctx context.Context, formID int, req CreateWebhookRequest) (*WebhookConfig, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	path := fmt.Sprintf("/api/v1/system/%d/webhooks", formID)
	var out WebhookConfig
	if err := c.do(ctx, http.MethodPost, path, nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type UpdateWebhookRequest struct {
	FQDN    string  `json:"fqdn"`
	URL     *string `json:"url,omitempty"`
	Remarks *string `json:"remarks,omitempty"`
}

// UpdateWebhookConfig calls PATCH /api/v1/system/{fid}/webhooks/{webhookId}
// (admin). FQDN is required and must match the current destination URL FQDN.
func (c *Client) UpdateWebhookConfig(ctx context.Context, formID int, webhookID string, req UpdateWebhookRequest) (*WebhookConfig, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	path := fmt.Sprintf("/api/v1/system/%d/webhooks/%s", formID, url.PathEscape(webhookID))
	var out WebhookConfig
	if err := c.do(ctx, http.MethodPatch, path, nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteWebhookConfig calls DELETE /api/v1/system/{fid}/webhooks/{webhookId}
// (admin). The fqdn query parameter is required and must match the current
// destination URL FQDN. Returns the raw response body for diagnostic output.
func (c *Client) DeleteWebhookConfig(ctx context.Context, formID int, webhookID, fqdn string) (json.RawMessage, error) {
	path := fmt.Sprintf("/api/v1/system/%d/webhooks/%s", formID, url.PathEscape(webhookID))
	q := url.Values{}
	q.Set("fqdn", fqdn)
	var out json.RawMessage
	if err := c.do(ctx, http.MethodDelete, path, q, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type Approval struct {
	DocID            int      `json:"docid"`
	Hidden           *bool    `json:"hidden,omitempty"`
	Attachment       bool     `json:"attachment"`
	Comment          bool     `json:"comment"`
	Title1           string   `json:"title1"`
	Title2           string   `json:"title2"`
	FormName         string   `json:"form_name"`
	Status           string   `json:"status"`
	DisplayStatus    string   `json:"display_status"`
	ApplyDatetime    string   `json:"apply_datetime"`
	ApplyUser        string   `json:"apply_user"`
	ApprovalUser     []string `json:"approval_user"`
	LastAprvDatetime string   `json:"lastaprv_datetime"`
}

type ApprovalsListResponse struct {
	TotalCount   int        `json:"total_count"`
	ApprovalList []Approval `json:"approval_list"`
}

// ApprovalsListParams holds query parameters for GET /api/v1/approvals.
// Stat is required. Zero values for *int / string / *bool mean "omit".
type ApprovalsListParams struct {
	Stat          int    // required
	FormGroupID   *int   // fgid
	FormID        *int   // fid
	Step          *int   // step
	RecordNo      *int   // record_no
	GetLine       *int   // get_line
	ProxyUser     string // proxy_user
	Filter        string // filter
	ShowHiddenDoc *bool  // show_hidden_doc
}

func (p ApprovalsListParams) query() url.Values {
	v := url.Values{}
	v.Set("stat", strconv.Itoa(p.Stat))
	if p.FormGroupID != nil {
		v.Set("fgid", strconv.Itoa(*p.FormGroupID))
	}
	if p.FormID != nil {
		v.Set("fid", strconv.Itoa(*p.FormID))
	}
	if p.Step != nil {
		v.Set("step", strconv.Itoa(*p.Step))
	}
	if p.RecordNo != nil {
		v.Set("record_no", strconv.Itoa(*p.RecordNo))
	}
	if p.GetLine != nil {
		v.Set("get_line", strconv.Itoa(*p.GetLine))
	}
	if p.ProxyUser != "" {
		v.Set("proxy_user", p.ProxyUser)
	}
	if p.Filter != "" {
		v.Set("filter", p.Filter)
	}
	if p.ShowHiddenDoc != nil {
		v.Set("show_hidden_doc", strconv.FormatBool(*p.ShowHiddenDoc))
	}
	return v
}

// ListApprovals calls GET /api/v1/approvals.
func (c *Client) ListApprovals(ctx context.Context, p ApprovalsListParams) (*ApprovalsListResponse, error) {
	var out ApprovalsListResponse
	if err := c.do(ctx, http.MethodGet, "/api/v1/approvals", p.query(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type ApprovalsWaitStatus struct {
	Type  int    `json:"type"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type ApprovalsWaitItem struct {
	DocID      int    `json:"docid"`
	Name       string `json:"name"`
	Title      string `json:"title"`
	WriterName string `json:"writername"`
	WriteDate  string `json:"writedate"`
}

type ApprovalsWaitResponse struct {
	StatusList []ApprovalsWaitStatus `json:"status_list"`
	WaitList   []ApprovalsWaitItem   `json:"wait_list"`
}

// ApprovalsWaitParams holds query parameters for GET /api/v1/approvals/wait.
type ApprovalsWaitParams struct {
	FormGroupID *int
	FormID      *int
	Step        *int
}

func (p ApprovalsWaitParams) query() url.Values {
	v := url.Values{}
	if p.FormGroupID != nil {
		v.Set("fgid", strconv.Itoa(*p.FormGroupID))
	}
	if p.FormID != nil {
		v.Set("fid", strconv.Itoa(*p.FormID))
	}
	if p.Step != nil {
		v.Set("step", strconv.Itoa(*p.Step))
	}
	return v
}

// GetApprovalsWait calls GET /api/v1/approvals/wait.
func (c *Client) GetApprovalsWait(ctx context.Context, p ApprovalsWaitParams) (*ApprovalsWaitResponse, error) {
	var out ApprovalsWaitResponse
	if err := c.do(ctx, http.MethodGet, "/api/v1/approvals/wait", p.query(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SetApprovalsHiddenRequest is the PUT /api/v1/approvals/hidden body.
type SetApprovalsHiddenRequest struct {
	Hidden    bool   `json:"hidden"`
	DocID     []int  `json:"docid"`
	ProxyUser string `json:"proxy_user,omitempty"`
}

type SetApprovalsHiddenResponse struct {
	DocID       []int  `json:"docid"`
	MessageType int    `json:"message_type"`
	Message     string `json:"message"`
}

// SetApprovalsHidden calls PUT /api/v1/approvals/hidden.
func (c *Client) SetApprovalsHidden(ctx context.Context, req SetApprovalsHiddenRequest) (*SetApprovalsHiddenResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	var out SetApprovalsHiddenResponse
	if err := c.do(ctx, http.MethodPut, "/api/v1/approvals/hidden", nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type SearchForm struct {
	ID   int    `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type SearchRoute struct {
	Code string `json:"code,omitempty"`
	Name string `json:"name,omitempty"`
}

type SearchItem struct {
	DocID            int         `json:"docid"`
	HasAttachments   bool        `json:"has_attachments"`
	HasComments      bool        `json:"has_comments"`
	Title1           string      `json:"title1"`
	Title2           string      `json:"title2"`
	Form             SearchForm  `json:"form"`
	Route            SearchRoute `json:"route"`
	Step             int         `json:"step"`
	Stat             int         `json:"stat"`
	WriteDatetime    string      `json:"write_datetime"`
	UpdateDatetime   string      `json:"update_datetime"`
	Writer           string      `json:"writer"`
	CurrentApprovers []string    `json:"current_approvers"`
	URL              string      `json:"url"`
}

type SearchDocumentsResponse struct {
	TotalCount int          `json:"total_count"`
	Items      []SearchItem `json:"items"`
}

type SearchDocumentsParams struct {
	Size   *int
	Offset *int
	Page   *int
}

func (p SearchDocumentsParams) query() url.Values {
	v := url.Values{}
	if p.Size != nil {
		v.Set("size", strconv.Itoa(*p.Size))
	}
	if p.Offset != nil {
		v.Set("offset", strconv.Itoa(*p.Offset))
	}
	if p.Page != nil {
		v.Set("page", strconv.Itoa(*p.Page))
	}
	return v
}

// SearchDocuments calls POST /api/v1/search/documents.
// body is sent as the raw JSON request body; pass nil or {} to match all documents.
func (c *Client) SearchDocuments(ctx context.Context, p SearchDocumentsParams, body json.RawMessage) (*SearchDocumentsResponse, error) {
	if len(body) == 0 {
		body = json.RawMessage("{}")
	}
	var out SearchDocumentsResponse
	if err := c.do(ctx, http.MethodPost, "/api/v1/search/documents", p.query(), body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type CreateDocumentResponse struct {
	DocID       int    `json:"docid"`
	MessageType int    `json:"message_type"`
	Message     string `json:"message"`
}

// CreateDocument calls POST /api/v1/documents.
// body is sent as the raw JSON request body; it must contain route_code,
// datas, and a form identifier (form_code or form_name).
func (c *Client) CreateDocument(ctx context.Context, body json.RawMessage) (*CreateDocumentResponse, error) {
	if len(body) == 0 {
		return nil, fmt.Errorf("request body is required for document creation")
	}
	var out CreateDocumentResponse
	if err := c.do(ctx, http.MethodPost, "/api/v1/documents", nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetDocument calls GET /api/v1/documents/{docid}.
// The response varies by form, so it is returned as raw JSON and left for
// the caller to interpret (typically via --jq or a json output).
func (c *Client) GetDocument(ctx context.Context, docID int) (json.RawMessage, error) {
	path := fmt.Sprintf("/api/v1/documents/%d", docID)
	var out json.RawMessage
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type UpdateDocumentResponse struct {
	DocID       int    `json:"docid"`
	MessageType int    `json:"message_type"`
	Message     string `json:"message"`
}

// UpdateDocument calls PATCH /api/v1/documents/{docid}.
// body is the raw JSON request body; it may contain wf_type, datas,
// route_code, etc. route_code is typically required by the server.
func (c *Client) UpdateDocument(ctx context.Context, docID int, body json.RawMessage) (*UpdateDocumentResponse, error) {
	if len(body) == 0 {
		return nil, fmt.Errorf("request body is required for document update")
	}
	path := fmt.Sprintf("/api/v1/documents/%d", docID)
	var out UpdateDocumentResponse
	if err := c.do(ctx, http.MethodPatch, path, nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type DeleteDocumentResponse struct {
	MessageType int    `json:"message_type"`
	Message     string `json:"message"`
}

// DeleteDocument calls DELETE /api/v1/documents/{docid}.
func (c *Client) DeleteDocument(ctx context.Context, docID int) (*DeleteDocumentResponse, error) {
	path := fmt.Sprintf("/api/v1/documents/%d", docID)
	var out DeleteDocumentResponse
	if err := c.do(ctx, http.MethodDelete, path, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type SelfInfo struct {
	ID          string `json:"id"`
	UserName    string `json:"userName"`
	DisplayName string `json:"displayName"`
	// UserCode is the X-point user code from the atled SCIM extension
	// (urn:atled:scim:schemas:1.0:User.userCode). It is the identifier used
	// in writer_list and other X-point APIs — distinct from userName, which
	// is the login name.
	AtledExt struct {
		UserCode string `json:"userCode"`
	} `json:"urn:atled:scim:schemas:1.0:User"`
}

// GetSelfInfo calls GET /scim/v2/{domain_code}/Me to fetch the authenticated
// user's info. Requires OAuth2 bearer auth.
func (c *Client) GetSelfInfo(ctx context.Context, domainCode string) (*SelfInfo, error) {
	if domainCode == "" {
		return nil, fmt.Errorf("domain code is required for /scim/v2/{domain_code}/Me")
	}
	path := fmt.Sprintf("/scim/v2/%s/Me", url.PathEscape(domainCode))
	var out SelfInfo
	if err := c.doAccept(ctx, http.MethodGet, path, nil, nil, "application/scim+json", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type AdminRoleResponse struct {
	Role []string `json:"role"`
}

// GetAdminRole calls GET /api/v1/adminrole to fetch the authenticated user's
// admin role list. Returns an empty list for general users.
func (c *Client) GetAdminRole(ctx context.Context) (*AdminRoleResponse, error) {
	var out AdminRoleResponse
	if err := c.do(ctx, http.MethodGet, "/api/v1/adminrole", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type ProxyUser struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type ProxyEntry struct {
	Use   ProxyUser `json:"use"`
	Apply bool      `json:"apply"`
	Aprv  bool      `json:"aprv"`
}

type ProxyInfoResponse struct {
	Proxy []ProxyEntry `json:"proxy"`
}

// GetProxyInfo calls GET /api/v1/proxy to fetch the authenticated user's
// delegation info.
func (c *Client) GetProxyInfo(ctx context.Context) (*ProxyInfoResponse, error) {
	var out ProxyInfoResponse
	if err := c.do(ctx, http.MethodGet, "/api/v1/proxy", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type XpointServiceInfo struct {
	Version      string   `json:"version"`
	APILevel     int      `json:"api_level"`
	SingleDomain bool     `json:"single_domain"`
	Features     []string `json:"features"`
}

// GetServiceInfo calls GET /x/v1/service to fetch X-point version and feature
// info. This endpoint does not require authentication.
func (c *Client) GetServiceInfo(ctx context.Context) (*XpointServiceInfo, error) {
	var out XpointServiceInfo
	if err := c.do(ctx, http.MethodGet, "/x/v1/service", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type QueryForm struct {
	FID      int    `json:"fid"`
	FormName string `json:"form_name"`
}

type Query struct {
	QueryID       int         `json:"query_id"`
	QueryCode     string      `json:"query_code"`
	QueryName     string      `json:"query_name"`
	QueryType     string      `json:"query_type"`
	QueryTypeName string      `json:"query_type_name,omitempty"`
	Remarks       string      `json:"remarks"`
	FormCount     int         `json:"form_count"`
	FID           int         `json:"fid,omitempty"`
	FormName      string      `json:"form_name,omitempty"`
	Forms         []QueryForm `json:"forms,omitempty"`
}

type QueryGroup struct {
	QueryGroupID   int     `json:"query_group_id"`
	QueryGroupName string  `json:"query_group_name"`
	Queries        []Query `json:"queries"`
}

type QueryListResponse struct {
	QueryGroups []QueryGroup `json:"query_groups"`
}

// ListAvailableQueries calls GET /api/v1/query/ (ユーザー別API: 利用可能クエリ一覧取得).
func (c *Client) ListAvailableQueries(ctx context.Context) (*QueryListResponse, error) {
	var out QueryListResponse
	if err := c.do(ctx, http.MethodGet, "/api/v1/query/", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetQueryParams holds query parameters for GET /api/v1/query/{query_code}.
// A nil Rows or Offset means "omit and let the server use its default".
type GetQueryParams struct {
	ExecFlag bool // true -> exec_flg=true (executes the query)
	Rows     *int // rows (server default 500, range 1-10000)
	Offset   *int // offset (server default 0)
}

func (p GetQueryParams) query() url.Values {
	v := url.Values{}
	if p.ExecFlag {
		v.Set("exec_flg", "true")
	}
	if p.Rows != nil {
		v.Set("rows", strconv.Itoa(*p.Rows))
	}
	if p.Offset != nil {
		v.Set("offset", strconv.Itoa(*p.Offset))
	}
	return v
}

// GetQuery calls GET /api/v1/query/{query_code}. When ExecFlag is true the
// response includes exec_result. The response shape varies by query_type
// (list/summary/cross), so it is returned as raw JSON.
func (c *Client) GetQuery(ctx context.Context, queryCode string, p GetQueryParams) (json.RawMessage, error) {
	path := fmt.Sprintf("/api/v1/query/%s", url.PathEscape(queryCode))
	var out json.RawMessage
	if err := c.do(ctx, http.MethodGet, path, p.query(), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetQueryGraph calls GET /api/v1/query/graph/{query_code} and returns the
// image bytes and the server-provided filename (from Content-Disposition).
// outFormat must be "png" or "jpeg" (empty string defaults to png on the
// server).
func (c *Client) GetQueryGraph(ctx context.Context, queryCode, outFormat string) (string, []byte, error) {
	path := fmt.Sprintf("/api/v1/query/graph/%s", url.PathEscape(queryCode))
	var q url.Values
	if outFormat != "" {
		q = url.Values{}
		q.Set("outFormat", outFormat)
	}
	accept := "image/png"
	if outFormat == "jpeg" {
		accept = "image/jpeg"
	}
	return c.downloadBytes(ctx, http.MethodGet, path, q, nil, "", accept)
}

type AddCommentRequest struct {
	Content      string `json:"content"`
	AttentionFlg int    `json:"attentionflg"`
}

type CommentMutationResponse struct {
	DocID       int    `json:"docid"`
	Seq         int    `json:"seq"`
	MessageType int    `json:"message_type"`
	Message     string `json:"message"`
}

// AddComment calls POST /api/v1/documents/{docid}/comments.
func (c *Client) AddComment(ctx context.Context, docID int, req AddCommentRequest) (*CommentMutationResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	path := fmt.Sprintf("/api/v1/documents/%d/comments", docID)
	var out CommentMutationResponse
	if err := c.do(ctx, http.MethodPost, path, nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type Comment struct {
	SeqNo        string `json:"seqno"`
	AttentionFlg bool   `json:"attentionflg"`
	Content      string `json:"content"`
	WriterName   string `json:"writername"`
	WriteDate    string `json:"writedate"`
}

type GetCommentsResponse struct {
	DocID       int       `json:"docid"`
	CommentList []Comment `json:"comment_list"`
}

// GetComments calls GET /api/v1/documents/{docid}/comments.
func (c *Client) GetComments(ctx context.Context, docID int) (*GetCommentsResponse, error) {
	path := fmt.Sprintf("/api/v1/documents/%d/comments", docID)
	var out GetCommentsResponse
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateCommentRequest holds the PATCH body for a comment update. Pointers
// allow selectively updating only one of content / attentionflg.
type UpdateCommentRequest struct {
	Content      *string `json:"content,omitempty"`
	AttentionFlg *int    `json:"attentionflg,omitempty"`
}

// UpdateComment calls PATCH /api/v1/documents/{docid}/comments/{seq}.
func (c *Client) UpdateComment(ctx context.Context, docID, seq int, req UpdateCommentRequest) (*CommentMutationResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	path := fmt.Sprintf("/api/v1/documents/%d/comments/%d", docID, seq)
	var out CommentMutationResponse
	if err := c.do(ctx, http.MethodPatch, path, nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteComment calls DELETE /api/v1/documents/{docid}/comments/{seq}.
func (c *Client) DeleteComment(ctx context.Context, docID, seq int) (*CommentMutationResponse, error) {
	path := fmt.Sprintf("/api/v1/documents/%d/comments/%d", docID, seq)
	var out CommentMutationResponse
	if err := c.do(ctx, http.MethodDelete, path, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetDocumentStatus calls GET /api/v1/documents/{docid}/status.
// When history is true, approval histories for every version are included.
// The response shape is complex (form, status, step, flow_results, etc.),
// so it is returned as raw JSON for the caller to interpret.
func (c *Client) GetDocumentStatus(ctx context.Context, docID int, history bool) (json.RawMessage, error) {
	path := fmt.Sprintf("/api/v1/documents/%d/status", docID)
	var q url.Values
	if history {
		q = url.Values{}
		q.Set("history", "true")
	}
	var out json.RawMessage
	if err := c.do(ctx, http.MethodGet, path, q, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// DownloadPDF calls GET /api/v1/documents/{docid}/pdf and returns the PDF
// bytes and the server-provided filename (parsed from Content-Disposition,
// which may use RFC 5987 encoding). The filename is empty when the server
// does not provide one.
func (c *Client) DownloadPDF(ctx context.Context, docID int) (string, []byte, error) {
	path := fmt.Sprintf("/api/v1/documents/%d/pdf", docID)
	filename, body, err := c.downloadBytes(ctx, http.MethodGet, path, nil, nil, "", "application/pdf")
	return filename, body, err
}

// downloadBytes issues an HTTP request and returns the raw response bytes
// and the server-provided filename (from Content-Disposition). It is used
// for endpoints that return non-JSON payloads (PDF, HTML, etc.).
//
// accept is the Accept header value. contentType is the Content-Type of
// the request body; pass "" when body is nil or when Content-Type should
// be left unset (e.g. multipart where the caller sets the boundary).
func (c *Client) downloadBytes(ctx context.Context, method, path string, q url.Values, body []byte, contentType, accept string) (string, []byte, error) {
	filename, respBody, _, err := c.downloadBytesWithContentType(ctx, method, path, q, body, contentType, accept)
	return filename, respBody, err
}

// downloadBytesWithContentType is like downloadBytes but also returns the
// response Content-Type, useful for endpoints that switch format based on
// URL suffix (e.g. master data JSON/CSV).
func (c *Client) downloadBytesWithContentType(ctx context.Context, method, path string, q url.Values, body []byte, contentType, accept string) (string, []byte, string, error) {
	u := c.baseURL + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}

	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, reqBody)
	if err != nil {
		return "", nil, "", err
	}
	c.auth.apply(req)
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	debug := os.Getenv("XP_DEBUG") != ""
	if debug {
		fmt.Fprintf(os.Stderr, "[xp] %s %s\n", req.Method, u)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return "", nil, "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, "", fmt.Errorf("read response body: %w", err)
	}
	if debug {
		fmt.Fprintf(os.Stderr, "[xp] <- %s (%d bytes)\n", resp.Status, len(respBody))
		if resp.StatusCode/100 != 2 {
			fmt.Fprintf(os.Stderr, "[xp]    %s\n", string(respBody))
		}
	}
	if resp.StatusCode/100 != 2 {
		return "", nil, "", fmt.Errorf("xpoint api error: %s: %s", resp.Status, string(respBody))
	}
	return parseContentDispositionFilename(resp.Header.Get("Content-Disposition")), respBody, resp.Header.Get("Content-Type"), nil
}

// DocviewParams holds query parameters for GET /api/v1/documents/docview.
// Exactly one of FormCode or FormName must be provided. RouteCode is always
// required (empty string is a valid value for auto-select / standard forms).
type DocviewParams struct {
	FormCode  string // fcd
	FormName  string // formname
	RouteCode string // routecd (required, can be empty string)
	FromDocID *int   // fromdocid
	ProxyUser string // proxy_user
}

func (p DocviewParams) query() url.Values {
	v := url.Values{}
	if p.FormCode != "" {
		v.Set("fcd", p.FormCode)
	}
	if p.FormName != "" {
		v.Set("formname", p.FormName)
	}
	v.Set("routecd", p.RouteCode)
	if p.FromDocID != nil {
		v.Set("fromdocid", strconv.Itoa(*p.FromDocID))
	}
	if p.ProxyUser != "" {
		v.Set("proxy_user", p.ProxyUser)
	}
	return v
}

// GetDocview calls GET /api/v1/documents/docview and returns the HTML bytes.
func (c *Client) GetDocview(ctx context.Context, p DocviewParams) ([]byte, error) {
	_, body, err := c.downloadBytes(ctx, http.MethodGet, "/api/v1/documents/docview", p.query(), nil, "", "text/html")
	return body, err
}

// GetDocumentOpenview calls GET /api/v1/documents/{docid}/openview and
// returns the HTML bytes for viewing a document.
func (c *Client) GetDocumentOpenview(ctx context.Context, docID int, proxyUser string) ([]byte, error) {
	path := fmt.Sprintf("/api/v1/documents/%d/openview", docID)
	var q url.Values
	if proxyUser != "" {
		q = url.Values{}
		q.Set("proxy_user", proxyUser)
	}
	_, body, err := c.downloadBytes(ctx, http.MethodGet, path, q, nil, "", "text/html")
	return body, err
}

// GetDocumentStatusview calls GET /api/v1/documents/{docid}/statusview and
// returns the HTML bytes for the approval status view.
func (c *Client) GetDocumentStatusview(ctx context.Context, docID int) ([]byte, error) {
	path := fmt.Sprintf("/api/v1/documents/%d/statusview", docID)
	_, body, err := c.downloadBytes(ctx, http.MethodGet, path, nil, nil, "", "text/html")
	return body, err
}

// DocviewMultipartFile describes a file to attach to the multipart docview
// request. Name is the filename the server records; Remarks is the optional
// file description (備考). Content is the raw file bytes.
type DocviewMultipartFile struct {
	Name         string
	Remarks      string
	DetailNo     *int
	EvidenceType *int
	Content      []byte
}

// DocviewMultipartParams holds the multipart body of POST
// /multiapi/v1/documents/docview. Exactly one of FormCode or FormName must
// be provided. RouteCode is required.
type DocviewMultipartParams struct {
	FormCode  string
	FormName  string
	RouteCode string
	FromDocID *int
	ProxyUser string
	Datas     string // JSON-stringified pre-fill data
	File      *DocviewMultipartFile
}

// PostDocviewMultipart calls POST /multiapi/v1/documents/docview and
// returns the HTML bytes.
func (c *Client) PostDocviewMultipart(ctx context.Context, p DocviewMultipartParams) ([]byte, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	writeField := func(name, value string) error {
		if value == "" {
			return nil
		}
		return mw.WriteField(name, value)
	}
	if err := writeField("fcd", p.FormCode); err != nil {
		return nil, fmt.Errorf("write fcd: %w", err)
	}
	if err := writeField("formname", p.FormName); err != nil {
		return nil, fmt.Errorf("write formname: %w", err)
	}
	if err := mw.WriteField("routecd", p.RouteCode); err != nil {
		return nil, fmt.Errorf("write routecd: %w", err)
	}
	if p.FromDocID != nil {
		if err := mw.WriteField("fromdocid", strconv.Itoa(*p.FromDocID)); err != nil {
			return nil, fmt.Errorf("write fromdocid: %w", err)
		}
	}
	if err := writeField("proxy_user", p.ProxyUser); err != nil {
		return nil, fmt.Errorf("write proxy_user: %w", err)
	}
	if err := writeField("datas", p.Datas); err != nil {
		return nil, fmt.Errorf("write datas: %w", err)
	}
	if p.File != nil {
		h := textproto.MIMEHeader{}
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, p.File.Name))
		h.Set("Content-Type", "application/octet-stream")
		fw, err := mw.CreatePart(h)
		if err != nil {
			return nil, fmt.Errorf("create file part: %w", err)
		}
		if _, err := fw.Write(p.File.Content); err != nil {
			return nil, fmt.Errorf("write file content: %w", err)
		}
		if err := mw.WriteField("file_name", p.File.Name); err != nil {
			return nil, fmt.Errorf("write file_name: %w", err)
		}
		if err := writeField("remarks", p.File.Remarks); err != nil {
			return nil, fmt.Errorf("write remarks: %w", err)
		}
		if p.File.DetailNo != nil {
			if err := mw.WriteField("detail_no", strconv.Itoa(*p.File.DetailNo)); err != nil {
				return nil, fmt.Errorf("write detail_no: %w", err)
			}
		}
		if p.File.EvidenceType != nil {
			if err := mw.WriteField("evidence_type", strconv.Itoa(*p.File.EvidenceType)); err != nil {
				return nil, fmt.Errorf("write evidence_type: %w", err)
			}
		}
	}
	if err := mw.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	_, body, err := c.downloadBytes(ctx, http.MethodPost, "/multiapi/v1/documents/docview", nil, buf.Bytes(), mw.FormDataContentType(), "text/html")
	return body, err
}

// parseContentDispositionFilename extracts a filename from a Content-Disposition
// header value, preferring the RFC 5987 filename* form when present.
func parseContentDispositionFilename(cd string) string {
	if cd == "" {
		return ""
	}
	if _, params, err := mime.ParseMediaType(cd); err == nil {
		if name := params["filename"]; name != "" {
			return name
		}
	}
	return ""
}

// do executes an HTTP request and decodes a JSON response into out.
func (c *Client) do(ctx context.Context, method, path string, q url.Values, body []byte, out any) error {
	return c.doAccept(ctx, method, path, q, body, "application/json", out)
}

// doAccept is like do but lets the caller specify the Accept header value.
// SCIM endpoints, for instance, require "application/scim+json".
func (c *Client) doAccept(ctx context.Context, method, path string, q url.Values, body []byte, accept string, out any) error {
	u := c.baseURL + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}

	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, reqBody)
	if err != nil {
		return err
	}
	c.auth.apply(req)
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	debug := os.Getenv("XP_DEBUG") != ""
	if debug {
		fmt.Fprintf(os.Stderr, "[xp] %s %s\n", req.Method, u)
		for k, v := range req.Header {
			val := v[0]
			if k == "Authorization" {
				val = "Bearer " + redactToken(val[len("Bearer "):])
			}
			if k == "X-ATLED-Generic-API-Token" {
				val = redactToken(val)
			}
			fmt.Fprintf(os.Stderr, "[xp]   %s: %s\n", k, val)
		}
		if len(body) > 0 {
			fmt.Fprintf(os.Stderr, "[xp]   body: %s\n", string(body))
		}
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}
	if debug {
		fmt.Fprintf(os.Stderr, "[xp] <- %s (%d bytes)\n", resp.Status, len(respBody))
		if resp.StatusCode/100 != 2 {
			fmt.Fprintf(os.Stderr, "[xp]    %s\n", string(respBody))
		}
	}
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("xpoint api error: %s: %s", resp.Status, string(respBody))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
