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
	"context"
	"fmt"
	"net/http"
	"strings"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/condition"
	lang "configcenter/src/common/language"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/metadata"
	"configcenter/src/common/util"
	"configcenter/src/scene_server/host_server/logics"

	"github.com/gin-gonic/gin"
	"github.com/rentiansheng/xlsx"
)

// BuildExcelFromData product excel from data
func (lgc *Logics) BuildExcelFromData(ctx context.Context, objID string, fields map[string]Property, filter []string, data []mapstr.MapStr, xlsxFile *xlsx.File, header http.Header, modelBizID int64, usernameMap map[string]string, propertyList []string) error {
	rid := util.GetHTTPCCRequestID(header)

	ccLang := lgc.Language.CreateDefaultCCLanguageIf(util.GetLanguage(header))
	ccErr := lgc.CCErr.CreateDefaultCCErrorIf(util.GetLanguage(header))
	sheet, err := xlsxFile.AddSheet("inst")
	if err != nil {
		blog.Errorf("setExcelRowDataByIndex add excel sheet error, err:%s, rid:%s", err.Error(), rid)
		return err

	}
	// index=1 表格数据的起始索引，excel表格数据第一列为字段说明，第二列为数据列
	addSystemField(fields, common.BKInnerObjIDObject, ccLang, 1)

	if 0 == len(filter) {
		filter = getFilterFields(objID)
	} else {
		filter = append(filter, getFilterFields(objID)...)
	}

	instPrimaryKeyValMap := make(map[int64][]PropertyPrimaryVal)
	productExcelHeader(ctx, fields, filter, sheet, ccLang)
	// indexID := getFieldsIDIndexMap(fields)

	rowIndex := common.HostAddMethodExcelIndexOffset

	for _, rowMap := range data {

		instIDKey := metadata.GetInstIDFieldByObjID(objID)
		instID, err := rowMap.Int64(instIDKey)
		if err != nil {
			blog.Errorf("setExcelRowDataByIndex inst:%+v, not inst id key:%s, objID:%s, rid:%s", rowMap, instIDKey, objID, rid)
			return ccErr.Errorf(common.CCErrCommInstFieldNotFound, "instIDKey", objID)
		}
		// 使用中英文用户名重新构造用户列表(用户列表实际为逗号分隔的string型)
		rowMap, err = replaceEnName(rid, rowMap, usernameMap, propertyList, ccLang)
		if err != nil {
			blog.Errorf("rebuild user list field, rid: %s", rid)
			return err
		}

		primaryKeyArr := setExcelRowDataByIndex(rowMap, sheet, rowIndex, fields)

		instPrimaryKeyValMap[instID] = primaryKeyArr
		rowIndex++

	}

	err = lgc.BuildAssociationExcelFromData(ctx, objID, instPrimaryKeyValMap, xlsxFile, header, modelBizID)
	if err != nil {
		return err
	}
	return nil
}

// BuildHostExcelFromData product excel from data
func (lgc *Logics) BuildHostExcelFromData(ctx context.Context, objID string, fields map[string]Property,
	filter []string, data []mapstr.MapStr, xlsxFile *xlsx.File, header http.Header, modelBizID int64,
	usernameMap map[string]string, propertyList []string, objNames, objIDs []string) error {
	rid := util.ExtractRequestIDFromContext(ctx)
	ccLang := lgc.Language.CreateDefaultCCLanguageIf(util.GetLanguage(header))
	ccErr := lgc.CCErr.CreateDefaultCCErrorIf(util.GetLanguage(header))

	sheet, err := xlsxFile.AddSheet("host")
	if err != nil {
		blog.Errorf("add excel sheet failed, err: %v, rid: %s", err, rid)
		return err
	}

	extFieldsTopoID := "cc_ext_field_topo"
	extFieldsBizID := "cc_ext_biz"
	extFieldsModuleID := "cc_ext_module"
	extFieldsSetID := "cc_ext_set"
	extFieldKey := make([]string, 0)
	extFieldKey = append(extFieldKey, extFieldsTopoID, extFieldsBizID)

	extFields := map[string]string{
		extFieldsTopoID:   ccLang.Language("web_ext_field_topo"),
		extFieldsBizID:    ccLang.Language("biz_property_bk_biz_name"),
		extFieldsModuleID: ccLang.Language("web_ext_field_module_name"),
		extFieldsSetID:    ccLang.Language("web_ext_field_set_name"),
	}
	// 生成key,用于赋值遍历主机数据进行赋值
	for _, objID := range objIDs {
		extFieldKey = append(extFieldKey, "cc_ext_"+objID)
	}

	// 2 自定义层级名称在extFieldKey切片中起始位置为2，0,1索引为业务拓扑和业务名
	for idx, objName := range objNames {
		extFields[extFieldKey[idx+2]] = objName
	}

	extFieldKey = append(extFieldKey, extFieldsSetID, extFieldsModuleID)
	fields = addExtFields(fields, extFields, extFieldKey)
	// len(objNames)+5=tip + biztopo + biz + set + moudle + customLen, the former indexes is used by these columns
	addSystemField(fields, common.BKInnerObjIDHost, ccLang, len(objNames)+5)

	productHostExcelHeader(ctx, fields, filter, sheet, ccLang, objNames)

	instPrimaryKeyValMap := make(map[int64][]PropertyPrimaryVal)
	hanlehHostDataParam := &HanlehHostDataParam{
		HostData:             data,
		ExtFieldsTopoID:      extFieldsTopoID,
		ExtFieldsBizID:       extFieldsBizID,
		ExtFieldsModuleID:    extFieldsModuleID,
		ExtFieldsSetID:       extFieldsSetID,
		CcErr:                ccErr,
		ExtFieldKey:          extFieldKey,
		UsernameMap:          usernameMap,
		PropertyList:         propertyList,
		CcLang:               ccLang,
		Sheet:                sheet,
		Rid:                  rid,
		ObjID:                objID,
		ObjIDs:               objIDs,
		Fields:               fields,
		InstPrimaryKeyValMap: instPrimaryKeyValMap,
	}

	err = lgc.buildHostExcelData(hanlehHostDataParam)
	if err != nil {
		blog.Errorf("build host excel data failed, err: %v, rid: %s", err, rid)
		return err
	}

	err = lgc.BuildAssociationExcelFromData(ctx, objID, instPrimaryKeyValMap, xlsxFile, header, modelBizID)
	if err != nil {
		blog.Errorf("build association excel data failed, err: %v, rid: %s", err, rid)
		return err
	}
	return nil
}

// buildHostExcelData 处理主机数据，生成Excel表格数据
func (lgc *Logics) buildHostExcelData(hanlehHostDataParam *HanlehHostDataParam) error {
	rowIndex := common.HostAddMethodExcelIndexOffset
	for _, hostData := range hanlehHostDataParam.HostData {
		rowMap, err := mapstr.NewFromInterface(hostData[common.BKInnerObjIDHost])
		if err != nil {
			blog.Errorf("build host excel data failed, hostData: %#v, err: %v, rid: %s", hostData, err,
				hanlehHostDataParam.Rid)
			return hanlehHostDataParam.CcErr.CCError(common.CCErrCommReplyDataFormatError)
		}

		// handle custom extFieldKey,前两个元素为业务拓扑、业务，后两个元素为集群、模块，中间的为自定义层级列
		for idx, field := range hanlehHostDataParam.ExtFieldKey[2 : len(hanlehHostDataParam.ExtFieldKey)-2] {
			rowMap[field] = hostData[hanlehHostDataParam.ObjIDs[idx]]
		}
		rowMap[hanlehHostDataParam.ExtFieldsSetID] = hostData["sets"]
		rowMap[hanlehHostDataParam.ExtFieldsModuleID] = hostData["modules"]

		if _, exist := hanlehHostDataParam.Fields[common.BKCloudIDField]; exist {
			cloudAreaArr, err := rowMap.MapStrArray(common.BKCloudIDField)
			if err != nil {
				blog.Errorf("get cloud id failed, host: %#v, err: %v, rid: %s", hostData, err, hanlehHostDataParam.Rid)
				return hanlehHostDataParam.CcErr.CCError(common.CCErrCommReplyDataFormatError)
			}

			if len(cloudAreaArr) != 1 {
				blog.Errorf("host has many cloud areas, host: %#v, err: %v, rid: %s", hostData, err,
					hanlehHostDataParam.Rid)
				return hanlehHostDataParam.CcErr.CCError(common.CCErrCommReplyDataFormatError)
			}

			cloudArea := fmt.Sprintf("%v[%v]", cloudAreaArr[0][common.BKInstNameField],
				cloudAreaArr[0][common.BKInstIDField])
			rowMap.Set(common.BKCloudIDField, cloudArea)
		}

		moduleMap, ok := hostData[common.BKInnerObjIDModule].([]interface{})
		if ok {
			topos := util.GetStrValsFromArrMapInterfaceByKey(moduleMap, "TopModuleName")
			if len(topos) > 0 {
				idx := strings.Index(topos[0], logics.SplitFlag)
				if idx > 0 {
					rowMap[hanlehHostDataParam.ExtFieldsBizID] = topos[0][:idx]
				}

				toposNobiz := make([]string, 0)
				for _, topo := range topos {
					idx := strings.Index(topo, logics.SplitFlag)
					if idx > 0 && len(topo) >= idx+len(logics.SplitFlag) {
						toposNobiz = append(toposNobiz, topo[idx+len(logics.SplitFlag):])
					}
				}
				rowMap[hanlehHostDataParam.ExtFieldsTopoID] = strings.Join(toposNobiz, ", ")
			}
		}

		instIDKey := metadata.GetInstIDFieldByObjID(hanlehHostDataParam.ObjID)
		instID, err := rowMap.Int64(instIDKey)
		if err != nil {
			blog.Errorf("get inst id failed, inst: %#v, err: %v, rid: %s", rowMap, err, hanlehHostDataParam.Rid)
			return hanlehHostDataParam.CcErr.Errorf(common.CCErrCommInstFieldNotFound, instIDKey,
				hanlehHostDataParam.ObjID)
		}

		// 使用中英文用户名重新构造用户列表(用户列表实际为逗号分隔的string型)
		rowMap, err = replaceEnName(hanlehHostDataParam.Rid, rowMap, hanlehHostDataParam.UsernameMap,
			hanlehHostDataParam.PropertyList, hanlehHostDataParam.CcLang)
		if err != nil {
			blog.Errorf("rebuild user list field, err: %v, rid: %s", err, hanlehHostDataParam.Rid)
			return err
		}

		primaryKeyArr := setExcelRowDataByIndex(rowMap, hanlehHostDataParam.Sheet, rowIndex, hanlehHostDataParam.Fields)
		hanlehHostDataParam.InstPrimaryKeyValMap[instID] = primaryKeyArr
		rowIndex++
	}
	return nil
}

func (lgc *Logics) BuildAssociationExcelFromData(ctx context.Context, objID string, instPrimaryInfo map[int64][]PropertyPrimaryVal, xlsxFile *xlsx.File, header http.Header, modelBizID int64) error {
	defLang := lgc.Language.CreateDefaultCCLanguageIf(util.GetLanguage(header))
	rid := util.ExtractRequestIDFromContext(ctx)
	var instIDArr []int64
	for instID := range instPrimaryInfo {
		instIDArr = append(instIDArr, instID)
	}
	instAsst, err := lgc.fetchAssociationData(ctx, header, objID, instIDArr, modelBizID)
	if err != nil {
		return err
	}
	asstData, err := lgc.getAssociationData(ctx, header, objID, instAsst, modelBizID)
	if err != nil {
		return err
	}

	sheet, err := xlsxFile.AddSheet("assocation")
	if err != nil {
		blog.Errorf("setExcelRowDataByIndex add excel  assocation sheet error. err:%s, rid:%s", err.Error(), rid)
		return err
	}

	cond := &metadata.SearchAssociationObjectRequest{
		Condition: map[string]interface{}{
			condition.BKDBOR: []mapstr.MapStr{
				{
					common.BKObjIDField: objID,
				},
				{
					common.BKAsstObjIDField: objID,
				},
			},
		},
	}
	//确定关联标识的列表，定义excel选项下拉栏。此处需要查cc_ObjAsst表。
	resp, err := lgc.CoreAPI.TopoServer().Association().SearchObject(ctx, header, cond)
	if err != nil {
		blog.ErrorJSON("get object association list failed, err: %v, rid: %s", err, rid)
		return err
	}
	if err := resp.CCError(); err != nil {
		blog.ErrorJSON("get object association list failed, err: %v, rid: %s", resp.ErrMsg, rid)
		return err
	}
	asstList := resp.Data
	productExcelAssociationHeader(ctx, sheet, defLang, len(instAsst), asstList)

	rowIndex := common.HostAddMethodExcelAssociationIndexOffset

	for _, inst := range instAsst {
		sheet.Cell(rowIndex, 1).SetString(inst.ObjectAsstID)
		sheet.Cell(rowIndex, 2).SetString("")
		srcInst, ok := asstData[inst.ObjectID][inst.InstID]
		if !ok {
			blog.Warnf("BuildAssociationExcelFromData association inst:%+v, not inst id :%d, objID:%s, rid:%s", inst, inst.InstID, objID, rid)
			// return lgc.CCErr.CreateDefaultCCErrorIf(util.GetLanguage(header)).Errorf(common.CCErrCommInstDataNil, fmt.Sprintf("%s %d", objID, inst.InstID))
			continue
		}
		dstInst, ok := asstData[inst.AsstObjectID][inst.AsstInstID]
		if !ok {
			blog.Warnf("BuildAssociationExcelFromData association inst:%+v, not inst id :%d, objID:%s, rid:%s", inst, inst.InstID, inst.AsstObjectID, rid)
			continue
		}
		sheet.Cell(rowIndex, 3).SetString(buildEexcelPrimaryKey(srcInst))
		sheet.Cell(rowIndex, 4).SetString(buildEexcelPrimaryKey(dstInst))
		style := sheet.Cell(rowIndex, 3).GetStyle()
		style.Alignment.WrapText = true
		style = sheet.Cell(rowIndex, 4).GetStyle()
		style.Alignment.WrapText = true
		rowIndex++
	}

	return nil

}

func buildEexcelPrimaryKey(propertyArr []PropertyPrimaryVal) string {
	var contentArr []string
	for _, property := range propertyArr {
		contentArr = append(contentArr, buildExcelPrimaryStr(property))
	}
	return strings.Join(contentArr, common.ExcelAsstPrimaryKeySplitChar)
}

func buildExcelPrimaryStr(property PropertyPrimaryVal) string {
	return property.Name + common.ExcelAsstPrimaryKeyJoinChar + property.StrVal
}

// BuildExcelTemplate  return httpcode, error
func (lgc *Logics) BuildExcelTemplate(ctx context.Context, objID, filename string, header http.Header, defLang lang.DefaultCCLanguageIf, modelBizID int64) error {
	rid := util.GetHTTPCCRequestID(header)
	filterFields := getFilterFields(objID)
	// host excel template doesn't need export field bk_cloud_id
	if objID == common.BKInnerObjIDHost {
		filterFields = append(filterFields, common.BKCloudIDField)
	}

	fields, err := lgc.GetObjFieldIDs(objID, filterFields, nil, header, modelBizID,
		common.HostAddMethodExcelDefaultIndex)

	if err != nil {
		blog.Errorf("get %s fields error:%s, rid: %s", objID, err.Error(), rid)
		return err
	}

	var file *xlsx.File
	file = xlsx.NewFile()
	sheet, err := file.AddSheet(objID)
	if err != nil {
		blog.Errorf("get %s fields error: %v, rid: %s", objID, err, rid)
		return err
	}
	blog.V(5).Infof("BuildExcelTemplate fields count:%d, rid: %s", fields, rid)
	productExcelHeader(ctx, fields, filterFields, sheet, defLang)
	ProductExcelCommentSheet(ctx, file, defLang)

	if err = file.Save(filename); nil != err {
		blog.Errorf("save file failed, filename: %s, err: %+v, rid: %s", filename, err, rid)
		return err
	}

	return nil
}

func AddDownExcelHttpHeader(c *gin.Context, name string) {
	if strings.HasSuffix(name, ".xls") {
		c.Header("Content-Type", "application/vnd.ms-excel")
	} else {
		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	}
	c.Header("Accept-Ranges", "bytes")
	c.Header("Content-Disposition", "attachment; filename="+name) // 文件名
	c.Header("Cache-Control", "must-revalidate, post-check=0, pre-check=0")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
}

// GetExcelData excel数据，一个kv结构，key行数（excel中的行数），value内容
func GetExcelData(ctx context.Context, sheet *xlsx.Sheet, fields map[string]Property, defFields common.KvMap, isCheckHeader bool, firstRow int, defLang lang.DefaultCCLanguageIf) (map[int]map[string]interface{}, []string, error) {

	var err error
	nameIndexMap, err := checkExcelHeader(ctx, sheet, fields, isCheckHeader, defLang)
	if nil != err {
		return nil, nil, err
	}
	hosts := make(map[int]map[string]interface{})
	index := headerRow
	if 0 != firstRow {
		index = firstRow
	}
	errMsg := make([]string, 0)
	rowCnt := len(sheet.Rows)
	for ; index < rowCnt; index++ {
		row := sheet.Rows[index]
		host, getErr := getDataFromByExcelRow(ctx, row, index, fields, defFields, nameIndexMap, defLang)
		if 0 != len(getErr) {
			errMsg = append(errMsg, getErr...)
			continue
		}
		if 0 != len(host) {
			hosts[index+1] = host
		}
	}
	if 0 != len(errMsg) {
		return nil, errMsg, nil
	}

	return hosts, nil, nil

}

// GetExcelData excel数据，一个kv结构，key行数（excel中的行数），value内容
func GetRawExcelData(ctx context.Context, sheet *xlsx.Sheet, defFields common.KvMap, firstRow int, defLang lang.DefaultCCLanguageIf) (map[int]map[string]interface{}, []string, error) {

	var err error
	nameIndexMap, err := checkExcelHeader(ctx, sheet, nil, false, defLang)
	if nil != err {
		return nil, nil, err
	}
	hosts := make(map[int]map[string]interface{})
	index := headerRow
	if 0 != firstRow {
		index = firstRow
	}
	errMsg := make([]string, 0)
	rowCnt := len(sheet.Rows)
	for ; index < rowCnt; index++ {
		row := sheet.Rows[index]
		host, getErr := getDataFromByExcelRow(ctx, row, index, nil, defFields, nameIndexMap, defLang)
		if nil != getErr {
			errMsg = append(errMsg, getErr...)
			continue
		}
		if 0 == len(host) {
			hosts[index+1] = nil
		} else {
			hosts[index+1] = host
		}
	}
	if 0 != len(errMsg) {
		return nil, errMsg, nil
	}

	return hosts, nil, nil

}

func GetAssociationExcelData(sheet *xlsx.Sheet, firstRow int) map[int]metadata.ExcelAssociation {

	rowCnt := len(sheet.Rows)
	index := firstRow

	asstInfoArr := make(map[int]metadata.ExcelAssociation, 0)
	for ; index < rowCnt; index++ {
		row := sheet.Rows[index]
		op := row.Cells[associationOPColIndex].String()
		if op == "" {
			continue
		}

		asstObjID := row.Cells[assciationAsstObjIDIndex].String()
		srcInst := row.Cells[assciationSrcInstIndex].String()
		dstInst := row.Cells[assciationDstInstIndex].String()
		asstInfoArr[index] = metadata.ExcelAssociation{
			ObjectAsstID: asstObjID,
			Operate:      getAssociationExcelOperateFlag(op),
			SrcPrimary:   srcInst,
			DstPrimary:   dstInst,
		}
	}

	return asstInfoArr
}

// GetFilterFields 不需要展示字段
func GetFilterFields(objID string) []string {
	return getFilterFields(objID)
}

// GetCustomFields 用户展示字段export时优先排序
func GetCustomFields(filterFields []string, customFieldsStr string) []string {
	return getCustomFields(filterFields, customFieldsStr)
}

func getAssociationExcelOperateFlag(op string) metadata.ExcelAssociationOperate {
	opFlag := metadata.ExcelAssociationOperateError
	switch op {
	case associationOPAdd:
		opFlag = metadata.ExcelAssociationOperateAdd
	case associationOPDelete:
		opFlag = metadata.ExcelAssociationOperateDelete
	}

	return opFlag
}
