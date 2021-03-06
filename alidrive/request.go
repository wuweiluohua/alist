package alidrive

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Xhofe/alist/conf"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

func GetFile(fileId string) (*File, error) {
	url:=conf.Conf.AliDrive.ApiUrl+"/file/get"
	req:=GetReq{
		DriveId:               User.DefaultDriveId,
		FileId:                fileId,
		ImageThumbnailProcess: conf.ImageThumbnailProcess,
		VideoThumbnailProcess: conf.VideoThumbnailProcess,
	}
	var file File
	if body, err := DoPost(url, req,true); err != nil {
		log.Errorf("doPost出错:%s",err.Error())
		return nil,err
	}else {
		if err = json.Unmarshal(body,&file);err !=nil {
			log.Errorf("解析json[%s]出错:%s",string(body),err.Error())
			return nil,err
		}
	}
	if file.IsAvailable() {
		return &file,nil
	}
	if file.Code==conf.AccessTokenInvalid {
		if RefreshToken() {
			return GetFile(fileId)
		}
	}
	return nil,fmt.Errorf(file.Message)
}

func Search(key string,limit int, marker string) (*Files, error) {
	url:=conf.Conf.AliDrive.ApiUrl+"/file/search"
	req:=SearchReq{
		DriveId:               User.DefaultDriveId,
		ImageThumbnailProcess: conf.ImageThumbnailProcess,
		ImageUrlProcess:       conf.ImageUrlProcess,
		Limit:                 limit,
		Marker:                marker,
		OrderBy:               conf.OrderSearch,
		Query:                 fmt.Sprintf("name match '%s'",key),
		VideoThumbnailProcess: conf.VideoThumbnailProcess,
	}
	var files Files
	if body, err := DoPost(url, req,true); err != nil {
		log.Errorf("doPost出错:%s",err.Error())
		return nil,err
	}else {
		if err = json.Unmarshal(body,&files);err !=nil {
			log.Errorf("解析json[%s]出错:%s",string(body),err.Error())
			return nil,err
		}
	}
	if files.IsAvailable() {
		return &files,nil
	}
	if files.Code==conf.AccessTokenInvalid {
		if RefreshToken() {
			return Search(key,limit,marker)
		}
	}
	return nil,fmt.Errorf(files.Message)
}

func GetRoot(limit int,marker string,orderBy string,orderDirection string) (*Files,error) {
	return GetList(conf.Conf.AliDrive.RootFolder,limit,marker,orderBy,orderDirection)
}

func GetList(parent string,limit int,marker string,orderBy string,orderDirection string) (*Files,error) {
	url:=conf.Conf.AliDrive.ApiUrl+"/file/list"
	req:=ListReq{
		DriveId:               User.DefaultDriveId,
		Fields:                "*",
		ImageThumbnailProcess: conf.ImageThumbnailProcess,
		ImageUrlProcess:       conf.ImageUrlProcess,
		Limit:                 limit,
		Marker:                marker,
		OrderBy:               orderBy,
		OrderDirection:        orderDirection,
		ParentFileId:          parent,
		VideoThumbnailProcess: conf.VideoThumbnailProcess,
	}
	var files Files
	if body, err := DoPost(url, req,true); err != nil {
		log.Errorf("doPost出错:%s",err.Error())
		return nil,err
	}else {
		if err = json.Unmarshal(body,&files);err !=nil {
			log.Errorf("解析json[%s]出错:%s",string(body),err.Error())
			return nil,err
		}
	}
	if files.IsAvailable() {
		return &files,nil
	}
	if files.Code==conf.AccessTokenInvalid {
		if RefreshToken() {
			return GetRoot(limit,marker,orderBy,orderDirection)
		}
	}
	return nil,fmt.Errorf(files.Message)
}

func GetUserInfo() (*UserInfo,error) {
	url:=conf.Conf.AliDrive.ApiUrl+"/user/get"
	var user UserInfo
	if body, err := DoPost(url, map[string]interface{}{},true); err != nil {
		log.Errorf("doPost出错:%s",err.Error())
		return nil,err
	}else {
		if err = json.Unmarshal(body,&user);err !=nil {
			log.Errorf("解析json[%s]出错:%s",string(body),err.Error())
			return nil,err
		}
	}
	if user.IsAvailable() {
		return &user,nil
	}
	if user.Code==conf.AccessTokenInvalid {
		if RefreshToken() {
			return GetUserInfo()
		}
	}
	return nil,fmt.Errorf(user.Message)
}

func DoPost(url string,request interface{},auth bool) (body []byte, err error) {
	var(
		resp *http.Response
	)
	requestBody := new(bytes.Buffer)
	err = json.NewEncoder(requestBody).Encode(request)
	if err !=nil {
		log.Errorf("创建requestBody出错:%s",err.Error())
	}
	req,err:=http.NewRequest("POST",url,requestBody)
	log.Debugf("do_post_req:%v",req)
	if err != nil {
		log.Errorf("创建request出错:%s",err.Error())
		return
	}
	if auth {
		req.Header.Set("authorization",conf.Authorization)
	}
	req.Header.Add("content-type","application/json")
	req.Header.Add("user-agent","Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36")
	req.Header.Add("origin","https://aliyundrive.com")
	req.Header.Add("accept","*/*")
	req.Header.Add("Accept-Language", "zh-CN,zh;q=0.8,en-US;q=0.5,en;q=0.3")
	req.Header.Add("Connection", "keep-alive")

	for retryCount := 3; retryCount >= 0; retryCount-- {
		if resp,err=conf.Client.Do(req);err!=nil&&strings.Contains(err.Error(),"timeout") {
			<- time.After(time.Second)
		}else {
			break
		}
	}
	if err!=nil {
		log.Errorf("请求阿里云盘api时出错:%s",err.Error())
		return
	}
	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		log.Errorf("读取api返回内容失败")
	}
	return
}

func GetPaths(fileId string) (*[]Path,error) {
	paths:=make([]Path,0)
	for fileId != conf.Conf.AliDrive.RootFolder && fileId != "root" {
		file,err:=GetFile(fileId)
		if err !=nil {
			log.Errorf("获取path出错:%s",err.Error())
			return nil,err
		}
		paths=append(paths,Path{
			Name:   file.Name,
			FileId: file.FileId,
		})
		fileId=file.ParentFileId
	}
	paths=append(paths, Path{
		Name:   "Root",
		FileId: "root",
	})
	return &paths,nil
}