// Copyright (c) 2016 VMware, Inc. All Rights Reserved.
//
// This product is licensed to you under the Apache License, Version 2.0 (the "License").
// You may not use this product except in compliance with the License.
//
// This product may include a number of subcomponents with separate copyright notices and
// license terms. Your use of these subcomponents is subject to the terms and conditions
// of the subcomponent's license, as noted in the LICENSE file.

package manifest_test

import (
	. "github.com/vmware/photon-controller-cli/photon/manifest"

	. "github.com/vmware/photon-controller-cli/Godeps/_workspace/src/github.com/onsi/ginkgo"
	. "github.com/vmware/photon-controller-cli/Godeps/_workspace/src/github.com/onsi/gomega"
	"io/ioutil"
	"os"
)

var _ = Describe("Installation", func() {
	Describe("LoadInstallation", func() {
		var (
			file        *os.File
			fileContent string
		)

		JustBeforeEach(func() {
			var err error
			file, err = ioutil.TempFile("", "installation_")
			if err != nil {
				Fail("Could not create temporary test file.")
			}

			_, err = file.WriteString(fileContent)
			if err != nil {
				Fail("Could not write test file " + file.Name())
			}

			_ = file.Close()
		})

		AfterEach(func() {
			if file != nil {
				_ = os.Remove(file.Name())
				file = nil
			}
		})

		Describe("deployment", func() {
			Describe("iamge_datastores", func() {
				Context("when value is a comma separated list", func() {
					BeforeEach(func() {
						fileContent = `---
deployment:
  image_datastores: ds1, ds2
`
					})

					It("loads successfully", func() {
						inst, err := LoadInstallation(file.Name())
						Expect(err).To(BeNil())

						Expect(inst).ToNot(BeNil())
						Expect(inst.Deployment.ImageDatastores).To(BeEquivalentTo([]string{"ds1", "ds2"}))
					})
				})

				Context("when value is a string array", func() {
					BeforeEach(func() {
						fileContent = `---
deployment:
  image_datastores:
  - ds1
  - ds2
`
					})

					It("loads successfully", func() {
						inst, err := LoadInstallation(file.Name())
						Expect(err).To(BeNil())

						Expect(inst).ToNot(BeNil())
						Expect(inst.Deployment.ImageDatastores).To(BeEquivalentTo([]string{"ds1", "ds2"}))
					})
				})

				Context("when value is missing", func() {
					BeforeEach(func() {
						fileContent = `---
deployment:
`
					})

					It("loads successfully", func() {
						inst, err := LoadInstallation(file.Name())
						Expect(err).To(BeNil())

						Expect(inst).ToNot(BeNil())
						Expect(inst.Deployment.ImageDatastores).To(BeNil())
					})
				})
			})

			Describe("resume_system", func() {
				Context("when value is 'true'", func() {
					BeforeEach(func() {
						fileContent = `---
deployment:
  resume_system: true
`
					})

					It("loads successfully", func() {
						inst, err := LoadInstallation(file.Name())
						Expect(err).To(BeNil())

						Expect(inst).ToNot(BeNil())
						Expect(inst.Deployment.ResumeSystem).To(BeTrue())
					})
				})

				Context("when value is 'false'", func() {
					BeforeEach(func() {
						fileContent = `---
deployment:
  resume_system: false
`
					})

					It("loads successfully", func() {
						inst, err := LoadInstallation(file.Name())
						Expect(err).To(BeNil())

						Expect(inst).ToNot(BeNil())
						Expect(inst.Deployment.ResumeSystem).To(BeFalse())
					})
				})

				Context("when value is a value other than 'true' or 'false'", func() {
					BeforeEach(func() {
						fileContent = `---
deployment:
  resume_system: other_value
`
					})

					It("fails to load file", func() {
						inst, err := LoadInstallation(file.Name())
						Expect(err).To(MatchError(
							"yaml: unmarshal errors:\n  line 3: cannot unmarshal !!str `other_v...` into bool"))

						Expect(inst).To(BeNil())
					})
				})

				Context("when value is not provided", func() {
					BeforeEach(func() {
						fileContent = `---
deployment:
`
					})

					It("loads successfully", func() {
						inst, err := LoadInstallation(file.Name())
						Expect(err).To(BeNil())

						Expect(inst).ToNot(BeNil())
						Expect(inst.Deployment.ResumeSystem).To(BeFalse())
					})
				})
			})
		})
	})
})
