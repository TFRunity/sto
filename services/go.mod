module github.com/AlexeBabanov/sto/services

go 1.27

// go.work действует только локально; в Docker сборка идёт с GOWORK=off,
// поэтому chassis подключается через replace (ADR-036).
replace github.com/AlexeBabanov/sto/chassis => ../chassis
