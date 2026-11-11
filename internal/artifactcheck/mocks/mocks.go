package mocks

import "context"

type FakeImageChecker struct {
	Existing map[string]bool
	Error    error
	Checked  []string
}

func (fake *FakeImageChecker) ImageExists(_ context.Context, repository, tag string) (bool, error) {
	reference := repository + ":" + tag
	fake.Checked = append(fake.Checked, reference)
	return fake.Existing[reference], fake.Error
}

type FakeObjectChecker struct {
	Existing map[string]bool
	Error    error
	Checked  []string
}

func (fake *FakeObjectChecker) ObjectExists(_ context.Context, bucket, key string) (bool, error) {
	reference := bucket + "/" + key
	fake.Checked = append(fake.Checked, reference)
	return fake.Existing[reference], fake.Error
}
