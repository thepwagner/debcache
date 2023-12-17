package dynamic_test

// func TestPackageList_Release(t *testing.T) {
// 	t.Parallel()
// 	const (
// 		main  = repo.Component("main")
// 		amd64 = repo.Architecture("amd64")
// 	)

// 	pkgs := dynamic.PackageList{}
// 	pkgs.Add(main, amd64, debian.Paragraph{
// 		"Package":        "test",
// 		"Source":         "test",
// 		"Version":        "1.0",
// 		"Installed-Size": "32652",
// 		"Maintainer":     "Test Testerson <test@test.com>",
// 		"Description":    "Fake package for testing generating package lists",
// 		"Size":           "27165284",
// 		"MD5sum":         "3f5fe197341e898623ee1bc752014d56",
// 		"SHA256":         "382d8d9b5e84ab8a925323db11ebdddc35f3104d659f63e989939f2dfe6d0ae5",
// 	})

// 	r, err := pkgs.Release()
// 	assert.NoError(t, err)
// 	assert.Equal(t, "Debian", r["Origin"])
// 	assert.Equal(t, "main", r["Components"])
// 	assert.Equal(t, "amd64", r["Architectures"])
// 	assert.NotEmpty(t, r["Date"])
// 	assert.NotEmpty(t, r["SHA256"])
// }
