/*
 * Copyright (c) 2020 Devtron Labs
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package router

import (
	"github.com/devtron-labs/devtron/api/restHandler"
	"github.com/gorilla/mux"
)

type MigrateDbRouter interface {
	InitMigrateDbRouter(migrateRouter *mux.Router)
}
type MigrateDbRouterImpl struct {
	migrateDbRestHandler restHandler.MigrateDbRestHandler
}

func NewMigrateDbRouterImpl(migrateDbRestHandler restHandler.MigrateDbRestHandler) *MigrateDbRouterImpl {
	return &MigrateDbRouterImpl{migrateDbRestHandler: migrateDbRestHandler}
}
func (impl MigrateDbRouterImpl) InitMigrateDbRouter(migrateRouter *mux.Router) {
	migrateRouter.Path("/db").
		HandlerFunc(impl.migrateDbRestHandler.SaveDbConfig).
		Methods("POST")
	migrateRouter.Path("/db").
		HandlerFunc(impl.migrateDbRestHandler.FetchAllDbConfig).
		Methods("GET")
	migrateRouter.Path("/db/{id}").
		HandlerFunc(impl.migrateDbRestHandler.FetchOneDbConfig).
		Methods("GET")
	migrateRouter.Path("/db").
		HandlerFunc(impl.migrateDbRestHandler.UpdateDbConfig).
		Methods("PUT")
	migrateRouter.Path("/db/autocomplete").
		HandlerFunc(impl.migrateDbRestHandler.FetchDbConfigForAutoComp).
		Methods("GET")
}
