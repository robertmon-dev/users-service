package registry

import (
	"net/http/httptest"

	"users-service/app/models/requests"
	"users-service/app/utils"

	"github.com/gin-gonic/gin"
)

type FunctionContainer *map[string]interface{}

func CreateTestContext(values map[string]interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	for key, val := range values {
		ctx.Set(key, val)
	}
	return ctx, w
}

func GinContextWithData(contextData map[string]interface{}) *gin.Context {
	ctx, _ := CreateTestContext(contextData)
	return ctx
}

var FunctionRegistry FunctionContainer

func init() {
	if FunctionRegistry == nil {
		FunctionRegistry = &map[string]interface{}{}
	}

	(*FunctionRegistry)["utils.CheckEmailFormat"] = utils.CheckEmailFormat
	(*FunctionRegistry)["utils.CheckPasswordFormat"] = utils.CheckPasswordFormat
	(*FunctionRegistry)["utils.CheckTokenFormat"] = utils.CheckTokenFormat
	(*FunctionRegistry)["utils.CheckPhoneFormat"] = utils.CheckPhoneFormat
	(*FunctionRegistry)["utils.CheckIBANFormat"] = utils.CheckIBANFormat
	(*FunctionRegistry)["utils.CheckSWIFTFormat"] = utils.CheckSWIFTFormat
	(*FunctionRegistry)["utils.IsSuccessStatusCode"] = utils.IsSuccessStatusCode
	(*FunctionRegistry)["utils.Contains"] = utils.Contains
	(*FunctionRegistry)["utils.ValidateRoles"] = utils.ValidateRoles
	(*FunctionRegistry)["utils.GenerateRandomCode"] = utils.GenerateRandomCode
	(*FunctionRegistry)["utils.GenerateRandomCodes"] = utils.GenerateRandomCodes
	(*FunctionRegistry)["utils.GetUserID"] = utils.GetUserID
	(*FunctionRegistry)["utils.BindRoles"] = utils.BindRoles
	(*FunctionRegistry)["utils.GetToken"] = utils.GetToken

	(*FunctionRegistry)["tests.GinContextWithData"] = GinContextWithData
	(*FunctionRegistry)["requests.Register.Valid"] = func(req *requests.Register) error {
		return req.Valid()
	}
}
