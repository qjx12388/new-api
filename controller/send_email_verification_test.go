package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// SendEmailVerification serves both sign-up verification and email binding for
// logged-in users. When registration is disabled, anonymous callers must be
// rejected at the gate, while authenticated users keep the email-bind flow.
func TestSendEmailVerificationRespectsRegisterEnabled(t *testing.T) {
	previousDB := model.DB
	previousType := common.MainDatabaseType()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}))
	model.DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)

	previousRegisterEnabled := common.RegisterEnabled
	previousRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	require.NoError(t, i18n.Init())

	t.Cleanup(func() {
		model.DB = previousDB
		common.SetMainDatabaseType(previousType)
		common.RegisterEnabled = previousRegisterEnabled
		common.RedisEnabled = previousRedisEnabled
	})

	disabledMessage := i18n.Translate(i18n.LangEn, i18n.MsgUserRegisterDisabled)

	cases := []struct {
		name            string
		registerEnabled bool
		authenticated   bool
		expectBlocked   bool
	}{
		{"registration disabled rejects anonymous", false, false, true},
		{"registration disabled allows logged-in email bind", false, true, false},
		{"registration enabled allows anonymous", true, false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			common.RegisterEnabled = tc.registerEnabled

			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/verification?email=gate-test@example.com", nil)
			if tc.authenticated {
				c.Set("id", 1)
			}

			SendEmailVerification(c)

			var response struct {
				Success bool   `json:"success"`
				Message string `json:"message"`
			}
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
			if tc.expectBlocked {
				assert.False(t, response.Success)
				assert.Equal(t, disabledMessage, response.Message)
			} else {
				// The gate must not fire; the request proceeds and only fails
				// later (e.g. SMTP is not configured in tests).
				assert.NotEqual(t, disabledMessage, response.Message)
			}
		})
	}
}
