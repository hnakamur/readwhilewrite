// Package readwhilewrite provides a reader and a writer
// which is used to read a file while the writer is writing
// to the same file.
//
// When readers get an EOF they waits further writes by
// the writer. The writer notify readers when it writes
// data.
package readwhilewrite
