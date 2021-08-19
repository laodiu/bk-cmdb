/*
 * Tencent is pleased to support the open source community by making 蓝鲸 available.
 * Copyright (C) 2017-2018 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package logics

import (
	"configcenter/src/ac/extensions"
	"configcenter/src/apimachinery"
	"configcenter/src/common/language"
	"configcenter/src/scene_server/topo_server/logics/inst"
	"configcenter/src/scene_server/topo_server/logics/model"
	"configcenter/src/scene_server/topo_server/logics/operation"
)

// Logics provides management interface for operations of model and instance and related resources like association
type Logics interface {
	ClassificationOperation() model.ClassificationOperationInterface
	SetOperation() inst.SetOperationInterface
	ObjectOperation() model.ObjectOperationInterface
	IdentifierOperation() operation.IdentifierOperationInterface
}

type logics struct {
	classification model.ClassificationOperationInterface
	set            inst.SetOperationInterface
	object         model.ObjectOperationInterface
	identifier     operation.IdentifierOperationInterface
}

// New create a logics manager
func New(client apimachinery.ClientSetInterface, authManager *extensions.AuthManager,
	languageIf language.CCLanguageIf) Logics {
	classificationOperation := model.NewClassificationOperation(client, authManager)
	setOperation := inst.NewSetOperation(client, languageIf)
	objectOperation := model.NewObjectOperation(client, authManager)
	IdentifierOperation := operation.NewIdentifier(client)

	return &logics{
		classification: classificationOperation,
		set:            setOperation,
		object:         objectOperation,
		identifier:     IdentifierOperation,
	}
}

func (l *logics) ClassificationOperation() model.ClassificationOperationInterface {
	return l.classification
}

func (l *logics) SetOperation() inst.SetOperationInterface {
	return l.set
}

func (l *logics) ObjectOperation() model.ObjectOperationInterface {
	return l.object
}

func (l *logics) IdentifierOperation() operation.IdentifierOperationInterface {
	return l.identifier
}
