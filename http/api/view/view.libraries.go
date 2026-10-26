package view

import (
	logiclibraries "tty.fail/mrus/korda/logic/root/libraries"
)

type ImportPathValidation struct {
	ImportPath string `json:"importPath"`
	IsValid    bool   `json:"isValid"`
	Message    string `json:"message,omitempty"`
}

type LibraryValidation struct {
	ImportPaths []ImportPathValidation `json:"importPaths"`
}

func NewLibraryValidation(results []logiclibraries.PathResult) LibraryValidation {
	out := LibraryValidation{ImportPaths: make([]ImportPathValidation, 0, len(results))}
	for _, r := range results {
		out.ImportPaths = append(out.ImportPaths, ImportPathValidation{ImportPath: r.ImportPath, IsValid: r.IsValid, Message: r.Message})
	}

	return out
}
