// Package audit records privileged staff actions to a tamper-evident log and
// exposes them for the admin audit view.
package audit

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/crypto"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/ctxkey"
)

// Entry is one recorded admin action.
type Entry struct {
	ID         int64      `json:"id"`
	ActorID    *uuid.UUID `json:"actor_id,omitempty"`
	ActorEmail *string    `json:"actor_email,omitempty"` // resolved on read
	ActorRole  *string    `json:"actor_role,omitempty"`
	Action     string     `json:"action"`
	Target     string     `json:"target"`
	StatusCode int        `json:"status_code"`
	IP         *string    `json:"ip,omitempty"`
	OccurredAt time.Time  `json:"occurred_at"`
}

type Recorder interface {
	Record(ctx context.Context, e Entry) error
	List(ctx context.Context, page, pageSize int) ([]Entry, int, error)
}

type postgresRecorder struct {
	db  *pgxpool.Pool
	enc *crypto.Encryptor
}

func NewRecorder(db *pgxpool.Pool, enc *crypto.Encryptor) Recorder {
	return &postgresRecorder{db: db, enc: enc}
}

func (r *postgresRecorder) Record(ctx context.Context, e Entry) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO admin_audit_log (actor_id, actor_role, action, target, status_code, ip)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		e.ActorID, e.ActorRole, e.Action, e.Target, e.StatusCode, e.IP)
	return err
}

// List returns recent entries with the actor's current email resolved by join.
func (r *postgresRecorder) List(ctx context.Context, page, pageSize int) ([]Entry, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM admin_audit_log`).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT a.id, a.actor_id, u.email_encrypted, a.actor_role, a.action, a.target,
		       a.status_code, a.ip, a.occurred_at
		FROM admin_audit_log a
		LEFT JOIN users u ON u.id = a.actor_id
		ORDER BY a.occurred_at DESC
		LIMIT $1 OFFSET $2`, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	entries := []Entry{}
	for rows.Next() {
		var e Entry
		var emailEnc []byte
		if err := rows.Scan(&e.ID, &e.ActorID, &emailEnc, &e.ActorRole,
			&e.Action, &e.Target, &e.StatusCode, &e.IP, &e.OccurredAt); err != nil {
			return nil, 0, err
		}
		if len(emailEnc) > 0 {
			if email, derr := r.enc.Decrypt(emailEnc); derr == nil && email != "" {
				e.ActorEmail = &email
			}
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return entries, total, nil
}

// Middleware records mutating admin requests (POST/PUT/PATCH/DELETE) after they
// complete, capturing the actor, target, and resulting status. Read requests
// are not logged. Recording failures are logged but never block the response.
func Middleware(rec Recorder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isMutation(r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			e := Entry{
				Action:     r.Method,
				Target:     r.URL.Path,
				StatusCode: ww.Status(),
			}
			if raw, ok := r.Context().Value(ctxkey.UserID).(string); ok && raw != "" {
				if uid, err := uuid.Parse(raw); err == nil {
					e.ActorID = &uid
				}
			}
			if role, ok := r.Context().Value(ctxkey.Role).(string); ok && role != "" {
				e.ActorRole = &role
			}
			if ip := r.RemoteAddr; ip != "" {
				e.IP = &ip
			}

			// Use a detached context so the write survives request cancellation.
			recCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := rec.Record(recCtx, e); err != nil {
				log.Printf("audit: failed to record %s %s: %v", e.Action, e.Target, err)
			}
		})
	}
}

func isMutation(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}
