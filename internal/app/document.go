package app

import (
	"database/sql"
	"encoding/json"
	"math"
	"net/http"
	"regexp"
	"strings"
)

type Element struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	X           float64   `json:"x"`
	Y           float64   `json:"y"`
	Width       float64   `json:"width"`
	Height      float64   `json:"height"`
	Rotation    float64   `json:"rotation"`
	GroupID     string    `json:"groupId,omitempty"`
	Shape       string    `json:"shape,omitempty"`
	Fill        string    `json:"fill,omitempty"`
	Border      string    `json:"border,omitempty"`
	BorderWidth float64   `json:"borderWidth,omitempty"`
	Text        string    `json:"text,omitempty"`
	FontSize    float64   `json:"fontSize,omitempty"`
	Bold        bool      `json:"bold,omitempty"`
	Italic      bool      `json:"italic,omitempty"`
	Align       string    `json:"align,omitempty"`
	Color       string    `json:"color,omitempty"`
	AssetID     string    `json:"assetId,omitempty"`
	Points      []float64 `json:"points,omitempty"`
	PenWidth    float64   `json:"penWidth,omitempty"`
}
type CanvasSettings struct {
	Background string  `json:"background"`
	GridSize   float64 `json:"gridSize"`
	Snap       bool    `json:"snap"`
}
type Document struct {
	SchemaVersion int            `json:"schemaVersion"`
	Elements      []Element      `json:"elements"`
	Settings      CanvasSettings `json:"settings"`
}

func emptyDocument() Document {
	return Document{SchemaVersion: 1, Elements: []Element{}, Settings: CanvasSettings{Background: "dots", GridSize: 20}}
}

var colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}([0-9a-fA-F]{2})?$`)

func validColor(c string) bool { return c == "transparent" || colorPattern.MatchString(c) }
func finite(v float64) bool    { return !math.IsNaN(v) && !math.IsInf(v, 0) }
func validateDocument(d Document) error {
	if d.SchemaVersion != 1 {
		return bad("Unsupported document format.")
	}
	if len(d.Elements) > 10000 {
		return bad("A page can contain at most 10,000 elements.")
	}
	if d.Settings.Background != "dots" && d.Settings.Background != "plain" {
		return bad("Invalid canvas background.")
	}
	if !finite(d.Settings.GridSize) || d.Settings.GridSize < 1 || d.Settings.GridSize > 1000 {
		return bad("Grid spacing must be between 1 and 1,000.")
	}
	ids := map[string]bool{}
	groups := map[string]bool{}
	for _, e := range d.Elements {
		if e.ID == "" || len(e.ID) > 128 || ids[e.ID] {
			return bad("Invalid or duplicate element ID.")
		}
		ids[e.ID] = true
		if len(e.GroupID) > 128 {
			return bad("Invalid group ID.")
		}
		if e.GroupID != "" {
			groups[e.GroupID] = true
		}
		for _, n := range []float64{e.X, e.Y, e.Width, e.Height, e.Rotation, e.BorderWidth, e.PenWidth, e.FontSize} {
			if !finite(n) || math.Abs(n) > 1e9 {
				return bad("Invalid element geometry.")
			}
		}
		if e.Width < .1 || e.Height < .1 || e.Width > 1e6 || e.Height > 1e6 {
			return bad("Element dimensions are out of range.")
		}
		switch e.Type {
		case "shape":
			if e.Shape != "rectangle" && e.Shape != "circle" && e.Shape != "star" {
				return bad("Unsupported shape.")
			}
			if !validColor(e.Fill) || !validColor(e.Border) || e.BorderWidth < 0 || e.BorderWidth > 100 {
				return bad("Invalid shape style.")
			}
		case "text":
			if len(e.Text) > 100000 || e.FontSize < 1 || e.FontSize > 2000 || !validColor(e.Color) || (e.Align != "left" && e.Align != "center" && e.Align != "right") {
				return bad("Invalid text style or size.")
			}
		case "image":
			if e.AssetID == "" || len(e.AssetID) > 128 {
				return bad("Image asset is missing.")
			}
		case "stroke":
			if len(e.Points) < 4 || len(e.Points) > 40000 || len(e.Points)%2 != 0 || e.PenWidth <= 0 || e.PenWidth > 100 || !validColor(e.Color) {
				return bad("Invalid pen stroke.")
			}
			for _, n := range e.Points {
				if !finite(n) || math.Abs(n) > 1e6 {
					return bad("Invalid drawing point.")
				}
			}
		default:
			return bad("Unsupported element type.")
		}
	}
	for id := range groups {
		if ids[id] {
			return bad("Group IDs must be separate from element IDs.")
		}
	}
	return nil
}
func validateAssets(tx *sql.Tx, owner, pageID string, d Document) error {
	seen := map[string]bool{}
	for _, e := range d.Elements {
		if e.Type != "image" || seen[e.AssetID] {
			continue
		}
		seen[e.AssetID] = true
		var n int
		if err := tx.QueryRow("SELECT count(*) FROM assets WHERE id=? AND owner_id=?", e.AssetID, owner).Scan(&n); err != nil {
			return err
		}
		if n != 1 {
			return problem(400, "asset_missing", "An image is unavailable. Reattach it before saving.")
		}
		if _, err := tx.Exec("INSERT OR IGNORE INTO page_assets(page_id,asset_id) VALUES(?,?)", pageID, e.AssetID); err != nil {
			return err
		}
	}
	return nil
}
func cloneDocument(raw string) (string, error) {
	var d Document
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		return "", err
	}
	groups := map[string]string{}
	for i := range d.Elements {
		d.Elements[i].ID = randomID()
		g := d.Elements[i].GroupID
		if g != "" {
			if groups[g] == "" {
				groups[g] = randomID()
			}
			d.Elements[i].GroupID = groups[g]
		}
	}
	data, err := json.Marshal(d)
	return string(data), err
}
func (a *App) getDocument(w http.ResponseWriter, r *http.Request) (any, error) {
	var raw string
	var revision, metadata int
	var name, workspace string
	err := a.DB.QueryRow(`SELECT p.document,p.revision,p.metadata_revision,p.name,p.workspace_id FROM pages p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=? AND w.owner_id=?`, r.PathValue("id"), who(r).ID).Scan(&raw, &revision, &metadata, &name, &workspace)
	if err != nil {
		return nil, err
	}
	return map[string]any{"document": json.RawMessage(raw), "revision": revision, "metadataRevision": metadata, "name": name, "workspaceId": workspace}, nil
}

type saveRequest struct {
	BaseRevision int      `json:"baseRevision"`
	Epoch        int64    `json:"epoch"`
	ClientID     string   `json:"clientId"`
	MutationID   string   `json:"mutationId"`
	Document     Document `json:"document"`
}

func (a *App) saveDocument(w http.ResponseWriter, r *http.Request) (any, error) {
	var v saveRequest
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	if len(v.MutationID) < 8 || len(v.MutationID) > 128 || len(v.ClientID) < 8 || len(v.ClientID) > 128 {
		return nil, bad("Invalid save identity.")
	}
	if err := validateDocument(v.Document); err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(v.Document)
	if len(raw) > 16<<20 {
		return nil, problem(413, "document_limit", "The page exceeds the 16 MB document limit.")
	}
	requestRaw, _ := json.Marshal(v)
	hash := digest(string(requestRaw))
	p := who(r)
	id := r.PathValue("id")
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var revision int
	if err = tx.QueryRow("SELECT p.revision FROM pages p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=? AND w.owner_id=?", id, p.ID).Scan(&revision); err != nil {
		return nil, err
	}
	var receiptHash string
	var receiptRev int
	err = tx.QueryRow("SELECT digest,revision FROM save_receipts WHERE page_id=? AND client_id=? AND mutation_id=? AND expires_at>?", id, v.ClientID, v.MutationID, now()).Scan(&receiptHash, &receiptRev)
	if err == nil {
		if hash != receiptHash {
			return nil, bad("Mutation ID was reused with different content.")
		}
		return map[string]any{"revision": receiptRev, "mutationId": v.MutationID}, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}
	var session, client sql.NullString
	var epoch, expires int64
	err = tx.QueryRow("SELECT session_id,client_id,epoch,expires_at FROM page_leases WHERE page_id=?", id).Scan(&session, &client, &epoch, &expires)
	if err != nil || session.String != p.SessionID || client.String != v.ClientID || epoch != v.Epoch || expires <= now() {
		return nil, problem(409, "lease_lost", "This page is now read-only. Save your local work as a separate page.")
	}
	if revision != v.BaseRevision {
		return nil, problem(409, "revision_conflict", "The page changed in another session. Save your local work as a separate page.")
	}
	if err = validateAssets(tx, p.ID, id, v.Document); err != nil {
		return nil, err
	}
	revision++
	if _, err = tx.Exec("UPDATE pages SET document=?,revision=?,updated_at=? WHERE id=?", string(raw), revision, now(), id); err != nil {
		return nil, err
	}
	if _, err = tx.Exec("INSERT INTO save_receipts(page_id,client_id,mutation_id,digest,revision,expires_at) VALUES(?,?,?,?,?,?)", id, v.ClientID, v.MutationID, hash, revision, now()+86400); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	a.publish(p.ID, "page", id)
	a.enqueueImageVariants(p.ID, v.Document)
	return map[string]any{"revision": revision, "mutationId": v.MutationID}, nil
}
func (a *App) recoveryCopy(w http.ResponseWriter, r *http.Request) (any, error) {
	var v struct {
		WorkspaceID string   `json:"workspaceId"`
		Name        string   `json:"name"`
		Document    Document `json:"document"`
	}
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	if err := validateDocument(v.Document); err != nil {
		return nil, err
	}
	if strings.TrimSpace(v.Name) == "" {
		v.Name = "Recovered page"
	}
	if err := validName(v.Name); err != nil {
		return nil, err
	}
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var wid string
	if err = tx.QueryRow("SELECT id FROM workspaces WHERE id=? AND owner_id=?", v.WorkspaceID, who(r).ID).Scan(&wid); err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(v.Document)
	cloned, err := cloneDocument(string(raw))
	if err != nil {
		return nil, err
	}
	id := randomID()
	if _, err = tx.Exec("INSERT INTO pages(id,workspace_id,name,position,document,revision,updated_at) SELECT ?,?,?,COALESCE(MAX(position),-1)+1,?,1,? FROM pages WHERE workspace_id=?", id, wid, v.Name, cloned, now(), wid); err != nil {
		return nil, err
	}
	if err = validateAssets(tx, who(r).ID, id, v.Document); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	a.publish(who(r).ID, "workspace", wid)
	return map[string]string{"id": id}, nil
}
func validName(s string) error {
	if strings.TrimSpace(s) == "" || len(s) > 200 {
		return bad("Use a name between 1 and 200 characters.")
	}
	return nil
}
