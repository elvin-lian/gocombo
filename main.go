package main

import (
	"os"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"crypto/md5"
	"os/exec"
	"regexp"
	"strings"
)

type projectJson struct{
	RootPath                     string `json:"rootPath"`
	AssetsFolderPrefix           string `json:"assetsFolderPrefix"`
	AssetsConfFile               string `json:"assetsConfFile"`
	AssetsConfOutputFile         string `json:"assetsConfOutputFile"`
	MinFilesOutputPrefix         string `json:"minFilesOutputPrefix"`
}

type assetsJson struct{
	JsFiles   []string
	CssFiles  []string
	JsCodes   string
	CssCodes  string
}

var project *projectJson

func main() {
	name := ""
	if len(os.Args) > 1 {
		name = os.Args[1]
	}else {
		panic("请输入配置文件名称")
	}

	filename := "./conf/" + name + ".json"
	file, err := os.Open(filename)
	if err != nil {
		panic("项目配置文件不存在: " + filename)
	}
	defer file.Close()

	project = new(projectJson)

	dec := json.NewDecoder(file)
	err = dec.Decode(project)

	if err != nil {
		panic("配置文件解析失败:" + err.Error())
		os.Exit(1)
	}

	minify()
}

func fullFillImgUrl(assertUrl, css string) string{
	var fillImgUrl = func(str string) string {
		if strings.Index(str, "http://") > -1 {
			return str
		}

		regL := regexp.MustCompile(`\(\s*['"]?`)
		str = regL.ReplaceAllString(str,`("`)

		regR := regexp.MustCompile(`['"]?\s*\)`)
		str = regR.ReplaceAllString(str,`")`)

		if strings.Index(str, `("/`) > -1 {
			return str
		}

		str = strings.Replace(str,`("`,`("`+assertUrl,1)
		return str
	}
	reg := regexp.MustCompile(`url\s*\(\s*['"]?(.*?)['"]?\s*\)`)
	css = reg.ReplaceAllStringFunc(css, fillImgUrl)
	return css
}

func readAssetsConf() (assets map[string]assetsJson) {
	assets = make(map[string]assetsJson)

	file, err := os.Open(project.AssetsConfFile)
	if err != nil {
		panic("Assets配置文件不存在: " + project.AssetsConfFile)
	}
	defer file.Close()

	dec := json.NewDecoder(file)
	err = dec.Decode(&assets)

	if err != nil {
		panic("Assets配置文件解析失败: " + project.AssetsConfFile + "\n" + err.Error())
		os.Exit(1)
	}
	return
}

func minify() {
	fmt.Println("begin")

	newAssets := make(map[string]assetsJson)

	assets := readAssetsConf()
	for key, item := range assets {
		fmt.Println("minify ", key)
		jsName := minifyJs(item)
		cssName := minifyCss(item)

		newItem := assetsJson{
			JsFiles: []string{jsName},
			CssFiles: []string{cssName},
		}
		newAssets[key] = newItem
	}

	writeNewAssets(newAssets)

	fmt.Println("end")
}

func writeNewAssets(newAssets map[string]assetsJson) {
	file, err := os.Create(project.AssetsConfOutputFile)
	if err != nil {
		panic("不能读写文件: " + project.AssetsConfOutputFile + err.Error())
	}

	enc := json.NewEncoder(file)
	enc.Encode(newAssets)
}

func minifyJs(item assetsJson) string {

	content := ""
	for _, filename := range item.JsFiles {
		content += getFileContent(project.RootPath+project.AssetsFolderPrefix+filename)
	}
	content += item.JsCodes;

	hash := md5Hash(content)
	outputFile := project.MinFilesOutputPrefix + "/" + hash + ".js"

	output := "./tmp/tmp.js"
	ioutil.WriteFile(output, []byte(content), os.ModePerm)

	cmd := exec.Command("uglifyjs", output, "-o", project.RootPath+project.AssetsFolderPrefix+outputFile)
	err := cmd.Run()
	if err != nil {
		fmt.Println("uglifyjs error: " + err.Error())
	}

	return outputFile;
}

func minifyCss(item assetsJson) string {

	content := ""
	for _, filename := range item.CssFiles {
		tmp := getFileContent(project.RootPath+project.AssetsFolderPrefix+filename)
		arr := strings.Split(filename,"/")
		arr1 := arr[0:len(arr)-1]
		path := strings.Join(arr1,"/") + "/"
		content += fullFillImgUrl(path, tmp)
	}
	content += item.CssCodes;

	hash := md5Hash(content)
	outputFile := project.MinFilesOutputPrefix + "/" + hash + ".css"

	output := "./tmp/tmp.css"
	ioutil.WriteFile(output, []byte(content), os.ModePerm)

	cmd := exec.Command("csso", "-i", output, "-o", project.RootPath+project.AssetsFolderPrefix+outputFile)
	err := cmd.Run()
	if err != nil {
		fmt.Println("csso error: " + err.Error())
	}

	return outputFile;
}

func getFileContent(file string) (str string) {
	b, err := ioutil.ReadFile(file)
	if err == nil {
		str = string(b)
	}else {
		fmt.Println(err.Error())
	}
	return str
}

func md5Hash(str string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(str)))
}
