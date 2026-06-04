package exitcheck_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/aikowocki/yandex-go-musthave-metrics/cmd/staticlint/exitcheck"
)

// TestExitCheck прогоняет анализатор exitcheck по тестовым пакетам в testdata.
// analysistest сверяет реальные диагностики с маркерами `// want` в исходниках:
//   - mainpkg — os.Exit в main репортится, в helper-функции игнорируется;
//   - nested  — os.Exit во вложенном замыкании внутри main репортится;
//   - aliased — импорт os под алиасом по-прежнему ловится (детект через TypesInfo);
//   - notmain — пакет не main, поэтому диагностик нет.
func TestExitCheck(t *testing.T) {
	analysistest.Run(
		t,
		analysistest.TestData(),
		exitcheck.Analyzer,
		"mainpkg", "nested", "aliased", "notmain",
	)
}
