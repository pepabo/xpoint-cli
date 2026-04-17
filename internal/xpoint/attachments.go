package xpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
)

type Attachment struct {
	ContentType       string `json:"content_type"`
	Seq               int    `json:"seq"`
	Name              string `json:"name"`
	Size              int    `json:"size"`
	Remarks           string `json:"remarks,omitempty"`
	DetailNo          *int   `json:"detail_no,omitempty"`
	EvidenceType      *int   `json:"evidence_type,omitempty"`
	EvidenceTypeLabel string `json:"evidence_type_label,omitempty"`
}

type AttachmentListResponse struct {
	Attachments []Attachment `json:"attachments"`
}

// ListAttachments calls GET /api/v1/attachments/{docid}.
func (c *Client) ListAttachments(ctx context.Context, docID int) (*AttachmentListResponse, error) {
	path := fmt.Sprintf("/api/v1/attachments/%d", docID)
	var out AttachmentListResponse
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetAttachment calls GET /api/v1/attachments/{docid}/{attach_seq} and
// returns the file bytes along with the server-provided filename parsed
// from Content-Disposition.
func (c *Client) GetAttachment(ctx context.Context, docID, seq int) (string, []byte, error) {
	path := fmt.Sprintf("/api/v1/attachments/%d/%d", docID, seq)
	u := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", nil, err
	}
	c.auth.apply(req)

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

// AddAttachmentRequest holds metadata for POST /multiapi/v1/attachments/{docid}.
// FileContent + FileName are required along with UserCode.
type AddAttachmentRequest struct {
	UserCode     string // required
	FileName     string // required; extension must be included
	FileContent  []byte // required
	Remarks      string
	Overwrite    *bool
	Reason       string
	DetailNo     *int
	EvidenceType *int
}

type AttachmentMutationResponse struct {
	DocID       int    `json:"docid"`
	Seq         int    `json:"seq"`
	MessageType int    `json:"message_type"`
	Message     string `json:"message"`
	Detail      string `json:"detail,omitempty"`
}

// AddAttachment calls POST /multiapi/v1/attachments/{docid} with a multipart body.
func (c *Client) AddAttachment(ctx context.Context, docID int, req AddAttachmentRequest) (*AttachmentMutationResponse, error) {
	if req.UserCode == "" || req.FileName == "" || len(req.FileContent) == 0 {
		return nil, fmt.Errorf("user_code, file_name, and file content are required")
	}
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	if err := mw.WriteField("user_code", req.UserCode); err != nil {
		return nil, err
	}
	if err := mw.WriteField("file_name", req.FileName); err != nil {
		return nil, err
	}
	fw, err := mw.CreateFormFile("file", req.FileName)
	if err != nil {
		return nil, err
	}
	if _, err := fw.Write(req.FileContent); err != nil {
		return nil, err
	}
	if req.Remarks != "" {
		if err := mw.WriteField("remarks", req.Remarks); err != nil {
			return nil, err
		}
	}
	if req.Overwrite != nil {
		if err := mw.WriteField("overwrite", strconv.FormatBool(*req.Overwrite)); err != nil {
			return nil, err
		}
	}
	if req.Reason != "" {
		if err := mw.WriteField("reason", req.Reason); err != nil {
			return nil, err
		}
	}
	if req.DetailNo != nil {
		if err := mw.WriteField("detail_no", strconv.Itoa(*req.DetailNo)); err != nil {
			return nil, err
		}
	}
	if req.EvidenceType != nil {
		if err := mw.WriteField("evidence_type", strconv.Itoa(*req.EvidenceType)); err != nil {
			return nil, err
		}
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/multiapi/v1/attachments/%d", docID)
	var out AttachmentMutationResponse
	if err := c.doMultipart(ctx, http.MethodPost, path, body.Bytes(), mw.FormDataContentType(), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateAttachmentRequest holds metadata for PATCH /multiapi/v1/attachments/{docid}/{attach_seq}.
// Set Delete=true to delete. When updating, FileContent/FileName are optional (omit to
// keep existing file); when deleting, file-related fields are ignored.
type UpdateAttachmentRequest struct {
	UserCode     string // required
	Delete       bool   // true -> delete the attachment
	FileName     string // required when FileContent is set
	FileContent  []byte
	Remarks      string
	Reason       string
	DetailNo     *int
	EvidenceType *int
}

// UpdateAttachment calls PATCH /multiapi/v1/attachments/{docid}/{attach_seq}. The same
// endpoint handles both updates and deletes (controlled by the delete form field).
func (c *Client) UpdateAttachment(ctx context.Context, docID, seq int, req UpdateAttachmentRequest) (*AttachmentMutationResponse, error) {
	if req.UserCode == "" {
		return nil, fmt.Errorf("user_code is required")
	}
	if !req.Delete && len(req.FileContent) > 0 && req.FileName == "" {
		return nil, fmt.Errorf("file_name is required when file content is provided")
	}
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	if err := mw.WriteField("user_code", req.UserCode); err != nil {
		return nil, err
	}
	if err := mw.WriteField("delete", strconv.FormatBool(req.Delete)); err != nil {
		return nil, err
	}
	if !req.Delete && len(req.FileContent) > 0 {
		if err := mw.WriteField("file_name", req.FileName); err != nil {
			return nil, err
		}
		fw, err := mw.CreateFormFile("file", req.FileName)
		if err != nil {
			return nil, err
		}
		if _, err := fw.Write(req.FileContent); err != nil {
			return nil, err
		}
	}
	if req.Remarks != "" {
		if err := mw.WriteField("remarks", req.Remarks); err != nil {
			return nil, err
		}
	}
	if req.Reason != "" {
		if err := mw.WriteField("reason", req.Reason); err != nil {
			return nil, err
		}
	}
	if req.DetailNo != nil {
		if err := mw.WriteField("detail_no", strconv.Itoa(*req.DetailNo)); err != nil {
			return nil, err
		}
	}
	if req.EvidenceType != nil {
		if err := mw.WriteField("evidence_type", strconv.Itoa(*req.EvidenceType)); err != nil {
			return nil, err
		}
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/multiapi/v1/attachments/%d/%d", docID, seq)
	var out AttachmentMutationResponse
	if err := c.doMultipart(ctx, http.MethodPatch, path, body.Bytes(), mw.FormDataContentType(), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// doMultipart issues a multipart/form-data request and decodes a JSON response.
func (c *Client) doMultipart(ctx context.Context, method, path string, body []byte, contentType string, out any) error {
	u := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, u, bytes.NewReader(body))
	if err != nil {
		return err
	}
	c.auth.apply(req)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", contentType)

	debug := os.Getenv("XP_DEBUG") != ""
	if debug {
		fmt.Fprintf(os.Stderr, "[xp] %s %s\n", req.Method, u)
		fmt.Fprintf(os.Stderr, "[xp]   Content-Type: %s (%d bytes)\n", contentType, len(body))
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
