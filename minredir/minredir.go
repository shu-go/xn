package minredir

import (
	"fmt"
	"net/http"
)

// CodeOAuth2Extractor exitracts `code` from OAuth2 HTTP response.
func CodeOAuth2Extractor(r *http.Request, resultChan chan string) bool {
	code := r.FormValue("code")
	resultChan <- code
	return (code != "")
}

// LaunchMinServer launches temporal HTTP server.
func LaunchMinServer(port int, extractor func(r *http.Request, resultChan chan string) bool, resultChan chan string) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ok := extractor(r, resultChan)
		/*
			code := r.FormValue("code")
			codeChan <- code
		*/

		var color string
		var icon string
		var result string
		if ok /* code != "" */ {
			//success
			color = "green"
			icon = "&#10003;"
			result = "Successfully authenticated!!"
		} else {
			//fail
			color = "red"
			icon = "&#10008;"
			result = "FAILED!"
		}
		disp := fmt.Sprintf(`<div><span style="font-size:xx-large; color:%s; border:solid thin %s;">%s</span> %s</div>`, color, color, icon, result)

		fmt.Fprintf(w, `
<html>
	<head><title>%s pomi</title></head>
	<body onload="open(location, '_self').close();"> <!-- Chrome won't let me close! -->
		%s
		<hr />
		<p>This is a temporal page.<br />Please close it.</p>
	</body>
</html>
`, icon, disp)
	})
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)

	return nil
}
