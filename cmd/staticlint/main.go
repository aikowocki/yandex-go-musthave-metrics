// Команда staticlint — это собственный multichecker для статического анализа
// исходного кода проекта.
//
// # Назначение
//
// multichecker объединяет в одном бинарном файле набор статических анализаторов
// и запускает их по указанным пакетам. Анализаторы построены на стандартном
// фреймворке golang.org/x/tools/go/analysis, поэтому утилита совместима с
// механизмом `go vet -vettool`.
//
// # Состав анализаторов
//
// multichecker включает несколько групп анализаторов:
//
//  1. Стандартные анализаторы пакета golang.org/x/tools/go/analysis/passes —
//     базовые проверки, аналогичные `go vet` (printf, structtag, shadow и др.).
//  2. Все анализаторы класса SA пакета staticcheck.io (honnef.co/go/tools) —
//     обнаружение программных ошибок: некорректное использование stdlib,
//     гонки, мёртвый код, неэффективные конструкции.
//  3. Выбранные анализаторы остальных классов staticcheck.io:
//     S (simple) — упрощение кода, ST (stylecheck) — стилевые проверки,
//     QF (quickfix) — рефакторинг-подсказки.
//  4. Публичные сторонние анализаторы:
//     - errcheck (github.com/kisielk/errcheck) — поиск необработанных ошибок;
//     - bodyclose (github.com/timakin/bodyclose) — поиск незакрытых
//     http.Response.Body.
//  5. Собственный анализатор exitcheck — запрещает прямой вызов os.Exit
//     в функции main пакета main.
//
// # Механизм запуска
//
// Сборка бинарного файла:
//
//	go build -o staticlint ./cmd/staticlint
//
// Запуск анализа всего проекта:
//
//	./staticlint ./...
//
// Запуск конкретного анализатора (например, только exitcheck):
//
//	./staticlint -exitcheck ./...
//
// Список всех доступных анализаторов и их флагов:
//
//	./staticlint help
//
// Подробная справка по конкретному анализатору:
//
//	./staticlint help exitcheck
//
// Также multichecker можно использовать как vettool:
//
//	go vet -vettool=$(pwd)/staticlint ./...
//
// Код считается прошедшим проверку, если ни один анализатор не выдал замечаний
// и процесс завершился с кодом 0.
package main

import (
	"strings"

	"github.com/kisielk/errcheck/errcheck"
	"github.com/timakin/bodyclose/passes/bodyclose"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/appends"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/quickfix"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"

	"github.com/aikowocki/yandex-go-musthave-metrics/cmd/staticlint/exitcheck"
)

// nonSAChecks перечисляет анализаторы классов S, ST и QF пакета staticcheck.io,
// которые включаются в multichecker дополнительно к полному набору класса SA.
// Ключ — имя анализатора (например, "S1000"), значение всегда true.
var nonSAChecks = map[string]bool{
	"S1000":  true, // simple: использовать обычный приём канала вместо select с одним case
	"S1002":  true, // simple: лишнее сравнение с булевой константой
	"S1011":  true, // simple: заменить цикл append на append с вариативным аргументом
	"ST1005": true, // stylecheck: строки ошибок не должны быть с заглавной буквы и без пунктуации
	"ST1016": true, // stylecheck: имя ресивера должно быть единообразным
	"QF1003": true, // quickfix: заменить if/else-if на switch
	"QF1012": true, // quickfix: использовать fmt.Fprintf вместо Write(fmt.Sprintf(...))
}

func main() {
	// checks хранит список анализаторов для multichecker.
	checks := []*analysis.Analyzer{
		// --- Стандартные анализаторы golang.org/x/tools/go/analysis/passes ---
		appends.Analyzer,       // проверяет, что у append есть значения для добавления
		asmdecl.Analyzer,       // сверяет ассемблерные объявления с Go-сигнатурами
		assign.Analyzer,        // находит бесполезные самоприсваивания
		atomic.Analyzer,        // ловит ошибочное использование sync/atomic
		bools.Analyzer,         // находит ошибки в булевых выражениях
		buildtag.Analyzer,      // проверяет корректность build-тегов
		cgocall.Analyzer,       // запрещает некорректные cgo-указатели
		composite.Analyzer,     // требует ключи в композитных литералах
		copylock.Analyzer,      // ловит копирование значений с мьютексами
		errorsas.Analyzer,      // проверяет корректность второго аргумента errors.As
		httpresponse.Analyzer,  // ловит ошибки работы с http.Response
		loopclosure.Analyzer,   // находит захват переменной цикла замыканием
		lostcancel.Analyzer,    // требует вызвать cancel у context.WithCancel
		nilfunc.Analyzer,       // ловит бессмысленные сравнения функций с nil
		printf.Analyzer,        // проверяет форматные строки Printf-подобных функций
		shadow.Analyzer,        // находит затенение переменных
		shift.Analyzer,         // ловит сдвиги, превышающие разрядность типа
		sigchanyzer.Analyzer,   // проверяет, что signal.Notify использует буферизированный канал
		stdmethods.Analyzer,    // сверяет сигнатуры методов стандартных интерфейсов
		stringintconv.Analyzer, // ловит подозрительные конверсии int в string
		structtag.Analyzer,     // проверяет корректность тегов структур
		tests.Analyzer,         // находит ошибки в сигнатурах тестов
		unmarshal.Analyzer,     // запрещает передавать не-указатель в Unmarshal
		unreachable.Analyzer,   // находит недостижимый код
		unsafeptr.Analyzer,     // ловит некорректное использование unsafe.Pointer
		unusedresult.Analyzer,  // требует использовать результат «чистых» функций

		// --- Сторонние публичные анализаторы ---
		errcheck.Analyzer,  // находит необработанные ошибки
		bodyclose.Analyzer, // находит незакрытый http.Response.Body

		// --- Собственный анализатор ---
		exitcheck.Analyzer, // запрещает os.Exit в func main пакета main
	}

	// Добавляем все анализаторы класса SA пакета staticcheck.
	for _, a := range staticcheck.Analyzers {
		if strings.HasPrefix(a.Analyzer.Name, "SA") {
			checks = append(checks, a.Analyzer)
		}
	}

	// Добавляем выбранные анализаторы остальных классов (S, ST, QF).
	// Все три пакета экспортируют []*lint.Analyzer, поэтому объединяем их.
	otherClasses := append(append(append([]*lint.Analyzer{}, simple.Analyzers...), stylecheck.Analyzers...), quickfix.Analyzers...)
	for _, a := range otherClasses {
		if nonSAChecks[a.Analyzer.Name] {
			checks = append(checks, a.Analyzer)
		}
	}

	multichecker.Main(checks...)
}
