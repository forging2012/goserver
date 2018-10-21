package upload

import (
	"strings"
	"github.com/gin-gonic/gin"
	"strconv"
	"time"
	"os"
	"github.com/gpmgo/gopm/modules/log"
	"io"
	"github.com/cihub/seelog"
	"goserver/webs"
	"goserver/utils"
)

func Upload(engine *gin.Engine, sp *webs.WebSp, bf webs.BaseFun, ele *utils.Element) {
	url := ele.MustAttr("Url")
	spName, spExt := ele.AttrValue("Sp")
	tmpDir := ele.Attr("TmpDir", "./temp/upload/")
	os.MkdirAll(tmpDir, 0777)
	DirUpload := ele.MustAttr("DirUpload")
	os.MkdirAll(DirUpload, 0777)
	var ImgWidth []int
	iw, iwb := ele.AttrValue("ImgMaxWidth")
	if iwb {
		iws := strings.Split(iw, ",")
		for _, w := range iws {
			wi, _ := strconv.Atoi(w)
			ImgWidth = append(ImgWidth, wi)
		}
	}
	seelog.Info("tmpDir=", tmpDir)
	UrlPrefix := ele.MustAttr("UrlPrefix")
	MainWidth, _ := strconv.Atoi(ele.Attr("MainWidth", "0"))

	engine.POST(url, func(ctx *gin.Context) {
		wb := webs.NewParam(ctx)
		ub := bf(wb).(*webs.UserBase)
		if ub != nil && ub.Result {
			mf, _ := ctx.MultipartForm()
			for k, _ := range mf.File {
				doUploadFileMd5(MainWidth, tmpDir, UrlPrefix, DirUpload, wb, k, ImgWidth...)
			}
			if sp != nil && spExt {
				sp.SpExec(spName, wb)
				ctx.JSON(200, wb.Out)
			} else {
				ctx.JSON(200, wb.Param)
			}
		} else {
			seelog.Error("upload file auth fail.")
			ctx.Status(401)
		}
	})
}

func doUploadFileMd5(MainWidth int, tmpDir, UrlPrefix, DirUpload string, c *webs.Param, key string, ImgWidth ... int) {
	tmp := strconv.FormatInt(time.Now().UnixNano(), 16)
	file, header, err := c.Context.Request.FormFile(key)

	filename := header.Filename
	ext := ".png"
	index := strings.Index(filename, ".")
	if index > 0 {
		ext = filename[index:]
	}
	tmpFileName := tmpDir + tmp + ext
	out, err := os.Create(tmpFileName)
	if err != nil {
		log.Error("上传文件创建临时文件夹失败!", tmpFileName, err)
		return
	}
	_, err2 := io.Copy(out, file)
	out.Close()
	file.Close()
	if err2 != nil {
		log.Error("写入上传文件流时失败!", tmpFileName, err)
		return
	}
	md5Name, e := Md5File(tmpFileName)
	if e == nil {
		pre, uri := utils.FmtImgDir(DirUpload+"/", md5Name)
		path := pre + ext
		if _, err := os.Stat(path); os.IsNotExist(err) {
			os.Rename(tmpFileName, path)
		} else {
			os.Remove(tmpFileName)
		}
		c.Param[key+"Prefix"] = pre
		if strings.ToLower(ext) != ".gif" && ImgWidth != nil {
			for _, w := range ImgWidth {
				ext8 := "_" + strconv.Itoa(w) + ext
				if _, err := os.Stat(pre + ext8); os.IsNotExist(err) {
					ImgResize(path, pre+ext8, w)
				}
				uri = uri + ext8
				if MainWidth == 0 || MainWidth == w || len(ImgWidth) == 1 {
					c.Param[key+"Url"] = UrlPrefix + uri + ext
				}
			}
		} else {
			c.Param[key+"Url"] = UrlPrefix + uri + ext
		}
	}
}
