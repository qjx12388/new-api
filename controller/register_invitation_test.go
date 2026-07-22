package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

func setupRegisterInvitationTest(t *testing.T) {
	t.Helper()
	previousDB := model.DB
	previousType := common.MainDatabaseType()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}))
	model.DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)

	previousRegisterEnabled := common.RegisterEnabled
	previousPasswordRegisterEnabled := common.PasswordRegisterEnabled
	previousEmailVerificationEnabled := common.EmailVerificationEnabled
	previousInvitationCodeRegisterEnabled := common.InvitationCodeRegisterEnabled
	previousQuotaForNewUser := common.QuotaForNewUser
	previousRedisEnabled := common.RedisEnabled
	common.RegisterEnabled = true
	common.PasswordRegisterEnabled = true
	common.EmailVerificationEnabled = false
	common.QuotaForNewUser = 0
	common.RedisEnabled = false
	require.NoError(t, i18n.Init())

	t.Cleanup(func() {
		model.DB = previousDB
		common.SetMainDatabaseType(previousType)
		common.RegisterEnabled = previousRegisterEnabled
		common.PasswordRegisterEnabled = previousPasswordRegisterEnabled
		common.EmailVerificationEnabled = previousEmailVerificationEnabled
		common.InvitationCodeRegisterEnabled = previousInvitationCodeRegisterEnabled
		common.QuotaForNewUser = previousQuotaForNewUser
		common.RedisEnabled = previousRedisEnabled
	})
}

func performRegisterRequest(t *testing.T, username, affCode string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := fmt.Sprintf(`{"username":%q,"password":"testpass123","aff_code":%q}`, username, affCode)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/user/register", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	Register(c)
	return recorder
}

func TestRegisterWithInvitationCodeRegisterEnabled(t *testing.T) {
	setupRegisterInvitationTest(t)
	common.InvitationCodeRegisterEnabled = true

	inviter := model.User{Username: "inviter", Password: "hashed-password", AffCode: "INVITE123"}
	require.NoError(t, model.DB.Create(&inviter).Error)

	t.Run("missing invitation code is rejected", func(t *testing.T) {
		recorder := performRegisterRequest(t, "user_no_code", "")
		var response struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		}
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
		assert.False(t, response.Success)
		assert.Equal(t, i18n.Translate(i18n.LangEn, i18n.MsgUserInvitationCodeRequired), response.Message)
	})

	t.Run("invalid invitation code is rejected", func(t *testing.T) {
		recorder := performRegisterRequest(t, "user_bad_code", "NO_SUCH_CODE")
		var response struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		}
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
		assert.False(t, response.Success)
		assert.Equal(t, i18n.Translate(i18n.LangEn, i18n.MsgUserInvitationCodeInvalid), response.Message)
	})

	t.Run("valid invitation code registers with inviter", func(t *testing.T) {
		recorder := performRegisterRequest(t, "user_with_code", "INVITE123")
		var response struct {
			Success bool `json:"success"`
		}
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
		require.True(t, response.Success, "register response: %s", recorder.Body.String())

		var created model.User
		require.NoError(t, model.DB.Where("username = ?", "user_with_code").First(&created).Error)
		assert.Equal(t, inviter.Id, created.InviterId)
	})
}

func TestRegisterWithInvitationCodeRegisterDisabled(t *testing.T) {
	setupRegisterInvitationTest(t)
	common.InvitationCodeRegisterEnabled = false

	t.Run("missing invitation code still registers", func(t *testing.T) {
		recorder := performRegisterRequest(t, "user_open", "")
		var response struct {
			Success bool `json:"success"`
		}
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
		require.True(t, response.Success, "register response: %s", recorder.Body.String())
	})

	t.Run("invalid invitation code is ignored", func(t *testing.T) {
		recorder := performRegisterRequest(t, "user_open_bad_code", "NO_SUCH_CODE")
		var response struct {
			Success bool `json:"success"`
		}
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
		require.True(t, response.Success, "register response: %s", recorder.Body.String())

		var created model.User
		require.NoError(t, model.DB.Where("username = ?", "user_open_bad_code").First(&created).Error)
		assert.Zero(t, created.InviterId)
	})
}
