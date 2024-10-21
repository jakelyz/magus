package main

import (
	"os"
	"reflect"
	"slices"
	"testing"
)

func TestGetPackages(t *testing.T) {
	files, _ := os.ReadDir("testdata/dotfiles")
	want := []Package{
		{name: "test-pkg"},
	}
	got := getPackages(files)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestGetFiles(t *testing.T) {
	test_path := "testdata/dotfiles/test-pkg"
	want := []string{".local/share/testfile", ".testpkgrc"}
	got := getFiles(test_path)

	if !slices.Equal(got, want) {
		t.Errorf("got %s want %s", got, want)
	}
}

func TestGetHash(t *testing.T) {
	test_string := []byte("test string")
	want := "6f8db599de986fab7a21625b7916589c"
	got := getHash(test_string)

	if got != want {
		t.Errorf("got %s want %s", got, want)
	}
}

func TestDetermineState(t *testing.T) {

	t.Run("determine present state", func(t *testing.T) {
		want := "PRESENT"
		got, _ := determineState(".testpkgrc", "f299060e0383392ebeac64b714eca7e3", "testdata/dotfiles/test-pkg")
		if got != want {
			t.Errorf("got %s, want %s", got, want)
		}
	})

	t.Run("determine absent state", func(t *testing.T) {
		want := "ABSENT"
		got, _ := determineState(".fakerc", "123456", "testdata/dotfiles/test-pkg")

		if got != want {
			t.Errorf("got %s, want %s", got, want)
		}
	})

	t.Run("determine mismatch state", func(t *testing.T) {
		want := "MISMATCH"
		got, _ := determineState(".testpkgrc", "123456789", "testdata/dotfiles/test-pkg")

		if got != want {
			t.Errorf("got %s, want %s", got, want)
		}
	})
	
}
