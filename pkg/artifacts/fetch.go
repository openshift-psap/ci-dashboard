package artifacts

import (
	"bytes"
	"crypto/md5"
    "encoding/hex"
	"encoding/json"

	"fmt"
	"io/ioutil"
    "net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	v1 "github.com/openshift-psap/ci-dashboard/api/matrix/v1"
	"github.com/PuerkitoBio/goquery"
)

type ArtifactType string

const (
	TypeJson ArtifactType      = "type:json"
	TypeJsonArray ArtifactType = "type:json-array"
	TypeHtml                   = "type:html"
	TypeBytes                  = "type:bytes"
	// Google GCP hosting doesn't return a 404 error when we request a
	// file that doesn't exist, but instead serves "an empty dir"
	// page.  From this page, we strip all the references to the path
	// we're requesting and compare it against this MD5sum (which was
	// printf-ed and reinjected here :#)
	MissingPageMD5Sum          = "b66c9aae6e6cf88de034b25232ba0181"
)

type ArtifactResult struct {
	Json JsonResult
	JsonArray JsonArray
	Html *goquery.Document
	Bytes []byte
}

type JsonResult map[string]interface{}
type JsonArray []interface{}

func fetchRemoveFromCache(test_matrix *v1.MatrixSpec, path string) error {
	cache_path := fmt.Sprintf("%s/%s", test_matrix.ArtifactsCache, path)
	return os.Remove(cache_path)
}

func IsPageNotFound(content []byte, path string) bool {
	fname_pos := strings.LastIndex(path,"/")
	content = []byte(strings.ReplaceAll(string(content), path, ""))
	content = []byte(strings.ReplaceAll(string(content), path[0:fname_pos+1], ""))

	hash := md5.Sum(content)
	hashString := hex.EncodeToString(hash[:])

	return hashString == MissingPageMD5Sum
}

func fetchArtifact(test_matrix *v1.MatrixSpec, path string) ([]byte, error) {
	cache_path := fmt.Sprintf("%s/%s", test_matrix.ArtifactsCache, path)
	artifact_url := fmt.Sprintf("%s/%s", test_matrix.ArtifactsURL, path)

	if strings.HasSuffix(cache_path, "/") {
		cache_path += "/?index"
	}

	content, err := ioutil.ReadFile(cache_path)
	if err == nil {
		if IsPageNotFound(content, path) {
			log.Debugf("File %s found in the cache, but 404", artifact_url)
			return content, fmt.Errorf("Page doesn't exist: %s", artifact_url)
		}

		log.Debugf("File %s found in the cache", artifact_url)
		return content, nil
	}

	log.Debugf("Fetching %s ...", artifact_url)
	resp, err := http.Get(artifact_url)
	if err != nil {
		return []byte{}, fmt.Errorf("error fetching %s: %v", artifact_url, err)
	}

	defer resp.Body.Close()
	content, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, fmt.Errorf("error reading %s: %v", artifact_url, err)
	}

	cache_dir, err := filepath.Abs(filepath.Dir(cache_path))
    if err != nil {
		log.Warningf("Failed to get cache directory for %s: %v", cache_path, err)
    }

	err = os.MkdirAll(cache_dir, os.ModePerm)
	if err != nil {
		log.Warningf("Failed to create cache directory %s: %v", cache_dir, err)
		return []byte{}, err
    }

	err = ioutil.WriteFile(cache_path, content, 0644)
	if err != nil {
		log.Warningf("Failed to write into cache file at %s: %v", cache_path, err)
	}

	if IsPageNotFound(content, path) {
		return content, fmt.Errorf("Page doesn't exist: %s", artifact_url)
	}

	return content, nil
}

func fetchHtmlArtifact(test_matrix *v1.MatrixSpec, path string) (*goquery.Document, error) {
	content, err := fetchArtifact(test_matrix, path)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("error parsing the HTML of %s: %v", path, err)
	}

	return doc, nil
}

func fetchJsonArtifact(test_matrix *v1.MatrixSpec, path string) (JsonResult, error){
	content, err := fetchArtifact(test_matrix, path)
	if err != nil {
		return nil, err
	}
	var result JsonResult
	err = json.Unmarshal(content, &result)
	if err != nil {
		fetchRemoveFromCache(test_matrix, path)
		return nil, fmt.Errorf("error parsing the JSON of %s: %v", path, err)
	}

	return result, nil
}

func fetchJsonArrayArtifact(test_matrix *v1.MatrixSpec, path string) (JsonArray, error){
	content, err := fetchArtifact(test_matrix, path)
	if err != nil {
		return nil, err
	}
	var result JsonArray
	err = json.Unmarshal(content, &result)
	if err != nil {
		//fetchRemoveFromCache(test_matrix, path)
		return nil, fmt.Errorf("error parsing the JSON of %s: %v", path, err)
	}

	return result, nil
}

func fetchTestResultResult(test_result *v1.TestResult, filename string, filetype ArtifactType) (ArtifactResult, error) {
	return fetchTestResult(test_result.TestSpec.Matrix, test_result.TestSpec.ProwName, test_result.BuildId, filename, filetype)
}

func fetchTestResult(test_matrix *v1.MatrixSpec, prow_name, build_id, filename string, filetype ArtifactType) (ArtifactResult, error) {
	file_path := fmt.Sprintf("%s/%s/%s", prow_name, build_id, filename)
	var result ArtifactResult
	var err error
	if filetype == TypeJson {
		result.Json, err = fetchJsonArtifact(test_matrix, file_path)
	} else if filetype == TypeJsonArray {
		result.JsonArray, err = fetchJsonArrayArtifact(test_matrix, file_path)
	} else if filetype == TypeHtml {
		result.Html, err = fetchHtmlArtifact(test_matrix, file_path)
	} else if filetype == TypeBytes {
		result.Bytes, err = fetchArtifact(test_matrix, file_path)
	} else {
		return result, fmt.Errorf("code error: invalid file type requested %s", filetype)
	}
	if err != nil {
		return result, fmt.Errorf("error fetching the test results from %s: %v", file_path, err)
	}

	return result, nil
}

func FetchLastTestResult(test_matrix *v1.MatrixSpec, test *v1.TestSpec, filename string, filetype ArtifactType) (string, ArtifactResult, error) {
	last_test_path := fmt.Sprintf("%s/latest-build.txt", test.ProwName)
	last_test_build_id, err := fetchArtifact(test_matrix, last_test_path)
	if err != nil {
		fetchRemoveFromCache(test_matrix, last_test_path)
		return "", ArtifactResult{}, fmt.Errorf("error fetching the last test build_id from %s: %v",
			last_test_path, err)
	}

	if _, err := strconv.Atoi(string(last_test_build_id)); err != nil {
		fetchRemoveFromCache(test_matrix, last_test_path)
		return "", ArtifactResult{}, fmt.Errorf("error validating the last test build_id from %s: %v",
			last_test_path, err)
	}

	if err = fetchRemoveFromCache(test_matrix, last_test_path); err != nil {
		log.Warningf("Failed to remove %s from cache : %v", last_test_path, err)
	}

	last_test_file, err := fetchTestResult(test_matrix, test.ProwName,
		string(last_test_build_id), filename, filetype)
	if (err != nil) {
		return "", ArtifactResult{}, fmt.Errorf("error fetching the results of the last test of %s: %s (%s): %v",
			test_matrix.Name, test.ProwName, last_test_build_id, err)

	}

	return string(last_test_build_id), last_test_file, nil
}

func FetchLastNTestResults(test_matrix *v1.MatrixSpec, prow_name string, test_history int, filename string, filetype ArtifactType) ([]string, map[string]ArtifactResult, error) {
	if test_history <= 0 {
		panic(fmt.Sprintf("Invalid number of test history required (%d)", test_history))
	}
	test_list_path := fmt.Sprintf("%s/", prow_name)
	test_list_html, err := fetchHtmlArtifact(test_matrix, test_list_path)
	if err != nil {
		return nil, nil, fmt.Errorf("error fetching the tests of %s / %s: %v", test_matrix.Name, prow_name, err)
	}

	test_results := map[string]ArtifactResult{}

	build_ids, err := ListFilesInDirectory(test_list_html, true, false)
	if err != nil {
		return nil, nil, fmt.Errorf("error fetching last test results: %v", err)
	}

	// `build_ids` order is "oldest first" (alphanumeric order of timestamps)
	if len(build_ids) > test_history {
		build_ids = build_ids[len(build_ids) - test_history:]
	}

	build_ids = reverseStringArray(build_ids)

	// `build_ids` order is now "newest first"

	for _, test_build_id := range build_ids {
		test_file, err := fetchTestResult(test_matrix, prow_name, test_build_id, filename, filetype)
		if (err != nil) {
			log.Warningf("error fetching the results of %s:%s/%s (%s): %v",
				test_matrix.Name, prow_name, test_build_id, filename, err)

		}

		test_results[test_build_id] = test_file
	}

	return build_ids, test_results, err
}

func FetchTestStepResult(test_result *v1.TestResult, filename string, filetype ArtifactType) (ArtifactResult, error) {
	var prow_step = test_result.TestSpec.Matrix.ProwStep
	if test_result.TestSpec.ProwStep != "" {
		// override test_matrix.ProwStep if ProwStep is test_spec.ProwStep is specified
		prow_step = test_result.TestSpec.ProwStep
	}
	step_filename := fmt.Sprintf("artifacts/%s/%s/%s", test_result.TestSpec.TestName, prow_step, filename)
	return fetchTestResultResult(test_result, step_filename, filetype)
}

func FetchTestToolboxSteps(test_result *v1.TestResult) ([]string, error) {
	html_toolbox_steps, err := FetchTestStepResult(test_result, "artifacts/", TypeHtml)
	if err != nil {
		return []string{}, err
	}

	toolbox_steps, err := ListFilesInDirectory(html_toolbox_steps.Html, true, false)
	if err != nil {
		return []string{}, fmt.Errorf("error fetching toolbox steps: %v", err)
	}

	return toolbox_steps, nil
}

func FetchTestWarnings(test_result *v1.TestResult) (map[string]string, error) {
	warning_dir := "artifacts/_WARNING"
	test_warnings_html, err := FetchTestStepResult(test_result, warning_dir, TypeHtml)
	if err != nil {
		return map[string]string{}, err
	}

	test_warning_files, err := ListFilesInDirectory(test_warnings_html.Html, false, true)
	if err != nil {
		return map[string]string{}, fmt.Errorf("error fetching toolbox steps: %v", err)
	}

	warnings := map[string]string{}
	for _, test_warning_filename := range test_warning_files {
		test_warning, err := FetchTestStepResult(test_result, warning_dir+"/"+test_warning_filename,
			TypeBytes)
		if err != nil {
			return map[string]string{}, err
		}
		warnings[test_warning_filename] = string(test_warning.Bytes)
	}
	return warnings, nil
}

func FetchTestToolboxLogs(test_result *v1.TestResult) (map[string]JsonArray, error) {
	toolbox_steps, err := FetchTestToolboxSteps(test_result)
	if err != nil {
		fmt.Println(err)
		return map[string]JsonArray{}, err
	}
	logs := map[string]JsonArray{}

	for _, toolbox_step := range toolbox_steps {
		ansible_log_path := "artifacts/"+ toolbox_step + "/_ansible.log.json"
		json_toolbox_step_logs, err := FetchTestStepResult(test_result, ansible_log_path, TypeJsonArray)
		if err != nil {
			log.Debugf("No logs for step %s: %v", toolbox_step, err)
			// no `_ansible.log.json` in the current step, meaning
			// that this directory wasn't generated by a
			// toolbox+ansible command. Ignore.
			continue
		}
		logs[toolbox_step] = json_toolbox_step_logs.JsonArray
		fmt.Println(toolbox_step)
	}

	return logs, nil
}

func reverseStringArray(arr []string) []string {
	for i := 0; i < len(arr)/2; i++ {
		j := len(arr) - i - 1
		arr[i], arr[j] = arr[j], arr[i]
	}
	return arr
}

func ListFilesInDirectory(html_dir *goquery.Document, dirs_only, files_only bool)([]string, error) {
	files := []string{}

	html_dir.Find("li.grid-row").EachWithBreak(func(i int, li_tag *goquery.Selection) bool {
		entry_type, found := li_tag.Find("img").Attr("src")

		if !found {
			// li-tag doesn't contain an img-tag, this is unexpected
			// in a directory listing.
			return true // continue
		}

		is_dir := entry_type == "/icons/dir.png"
		if files_only && is_dir || dirs_only && !is_dir {
			// li-tag is a directory entry, and we don't want directories.
			// or
			// li-tag is a file and we don't want files.
			return true // continue
		}

		filename := strings.TrimSuffix(strings.TrimSpace(li_tag.Find("a").Text()), "/")
		if filename == ".." {
			// skip "parent-dir" entry
			return true // continue
		}

		files = append(files, filename)

		return true
	})

	return files, nil
}
