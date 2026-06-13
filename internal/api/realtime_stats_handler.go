package api

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/nimbleflux/fluxbase/internal/database"
	apperrors "github.com/nimbleflux/fluxbase/internal/errors"
	"github.com/nimbleflux/fluxbase/internal/realtime"
)

func newRealtimeStatsHandler(db *database.Connection, manager *realtime.Manager) fiber.Handler {
	return func(c fiber.Ctx) error {
		role, _ := c.Locals("user_role").(string)
		if role != "admin" && role != "instance_admin" && role != "tenant_admin" && role != "service_role" && role != "tenant_service" {
			return SendForbidden(c, "Admin access required to view realtime stats", ErrCodeAccessDenied)
		}

		const defaultLimit = 25
		const maxLimit = 100
		limit := fiber.Query[int](c, "limit", defaultLimit)
		offset := fiber.Query[int](c, "offset", 0)
		search := strings.ToLower(c.Query("search", ""))

		limit, offset = NormalizePaginationParams(limit, offset, defaultLimit, maxLimit)

		allConnections := manager.GetConnectionsForStats()

		userIDs := make([]string, 0)
		for _, conn := range allConnections {
			if conn.UserID != nil {
				userIDs = append(userIDs, *conn.UserID)
			}
		}

		type userInfo struct {
			email       string
			displayName *string
		}
		userInfoMap := make(map[string]userInfo)
		if len(userIDs) > 0 {
			query := `SELECT id, email, raw_user_meta_data->>'display_name' as display_name FROM auth.users WHERE id = ANY($1)`
			rows, err := db.Query(c.RequestCtx(), query, userIDs)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var id, email string
					var displayName *string
					if err := rows.Scan(&id, &email, &displayName); err == nil {
						userInfoMap[id] = userInfo{
							email:       email,
							displayName: displayName,
						}
					}
				}
			}
		}

		enrichedConnections := make([]realtime.ConnectionInfo, 0, len(allConnections))
		for _, conn := range allConnections {
			if conn.UserID != nil {
				if info, ok := userInfoMap[*conn.UserID]; ok {
					conn.Email = &info.email
					conn.DisplayName = info.displayName
				}
			}
			enrichedConnections = append(enrichedConnections, conn)
		}

		var filteredConnections []realtime.ConnectionInfo
		if search != "" {
			for _, conn := range enrichedConnections {
				if strings.Contains(strings.ToLower(conn.ID), search) ||
					strings.Contains(strings.ToLower(conn.RemoteAddr), search) ||
					(conn.UserID != nil && strings.Contains(strings.ToLower(*conn.UserID), search)) ||
					(conn.Email != nil && strings.Contains(strings.ToLower(*conn.Email), search)) ||
					(conn.DisplayName != nil && strings.Contains(strings.ToLower(*conn.DisplayName), search)) {
					filteredConnections = append(filteredConnections, conn)
				}
			}
		} else {
			filteredConnections = enrichedConnections
		}

		total := len(filteredConnections)

		if offset >= len(filteredConnections) {
			filteredConnections = []realtime.ConnectionInfo{}
		} else {
			filteredConnections = filteredConnections[offset:]
		}
		if len(filteredConnections) > limit {
			filteredConnections = filteredConnections[:limit]
		}

		return apperrors.SendPaginated(c, "connections", filteredConnections, total, limit, offset)
	}
}
