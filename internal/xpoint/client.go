package xpoint

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strconv"
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

// DownloadPDF calls GET /api/v1/documents/{docid}/pdf and returns the PDF
// bytes and the server-provided filename (parsed from Content-Disposition,
// which may use RFC 5987 encoding). The filename is empty when the server
// does not provide one.
func (c *Client) DownloadPDF(ctx context.Context, docID int) (string, []byte, error) {
	path := fmt.Sprintf("/api/v1/documents/%d/pdf", docID)
	u := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", nil, err
	}
	c.auth.apply(req)
	req.Header.Set("Accept", "application/pdf")

	debug := os.Getenv("XP_DEBUG") != ""
	if debug {
		fmt.Fprintf(os.Stderr, "[xp] %s %s\n", req.Method, u)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("read response body: %w", err)
	}
	if debug {
		fmt.Fprintf(os.Stderr, "[xp] <- %s (%d bytes)\n", resp.Status, len(body))
		if resp.StatusCode/100 != 2 {
			fmt.Fprintf(os.Stderr, "[xp]    %s\n", string(body))
		}
	}
	if resp.StatusCode/100 != 2 {
		return "", nil, fmt.Errorf("xpoint api error: %s: %s", resp.Status, string(body))
	}
	return parseContentDispositionFilename(resp.Header.Get("Content-Disposition")), body, nil
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
