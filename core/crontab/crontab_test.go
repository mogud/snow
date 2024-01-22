package crontab

//
//func TestSuite(t *testing.T) {
//	RegisterFailHandler(Fail)
//	RunSpecs(t, "Crontab Test")
//}
//
//var _ = Describe("Crontab", func() {
//	Describe("Norm time", func() {
//		Context("With second configured", func() {
//			It("should be ok", func() {
//				desc := &CTDesc{second: []*ctDescUnit{{begin: 0, end: 59, step: 5}}}
//				desc.init()
//				Expect(desc.Norm(time.Date(2020, time.January, 5, 6, 7, 9, 0, time.Local)).Unix()).
//					Should(Equal(time.Date(2020, time.January, 5, 6, 7, 10, 0, time.Local).Unix()))
//
//				Expect(desc.Norm(time.Date(2020, time.January, 5, 6, 7, 8, 0, time.Local)).Unix()).
//					Should(Equal(time.Date(2020, time.January, 5, 6, 7, 10, 0, time.Local).Unix()))
//
//				Expect(desc.Norm(time.Date(2020, time.January, 5, 6, 7, 7, 0, time.Local)).Unix()).
//					Should(Equal(time.Date(2020, time.January, 5, 6, 7, 10, 0, time.Local).Unix()))
//
//				Expect(desc.Norm(time.Date(2020, time.January, 5, 6, 7, 6, 0, time.Local)).Unix()).
//					Should(Equal(time.Date(2020, time.January, 5, 6, 7, 10, 0, time.Local).Unix()))
//
//				Expect(desc.Norm(time.Date(2020, time.January, 5, 6, 7, 5, 0, time.Local)).Unix()).
//					Should(Equal(time.Date(2020, time.January, 5, 6, 7, 5, 0, time.Local).Unix()))
//
//				Expect(desc.Norm(time.Date(2020, time.January, 5, 6, 7, 56, 0, time.Local)).Unix()).
//					Should(Equal(time.Date(2020, time.January, 5, 6, 8, 0, 0, time.Local).Unix()))
//
//				Expect(desc.Norm(time.Date(2000, time.December, 31, 23, 59, 56, 0, time.Local)).Unix()).
//					Should(Equal(time.Date(2001, time.January, 1, 0, 0, 0, 0, time.Local).Unix()))
//
//				Expect(desc.Norm(time.Date(2000, time.December, 31, 23, 59, 59, 0, time.Local)).Unix()).
//					Should(Equal(time.Date(2001, time.January, 1, 0, 0, 0, 0, time.Local).Unix()))
//			})
//		})
//		Context("With minute configured", func() {
//			It("should be ok", func() {
//				desc := &CTDesc{minute: []*ctDescUnit{{begin: 3, end: 59, step: 7}}}
//				desc.init()
//				Expect(desc.Norm(time.Date(2020, time.January, 5, 6, 1, 0, 0, time.Local)).Unix()).
//					Should(Equal(time.Date(2020, time.January, 5, 6, 3, 0, 0, time.Local).Unix()))
//
//				Expect(desc.Norm(time.Date(2020, time.January, 5, 6, 3, 0, 1, time.Local)).Unix()).
//					Should(Equal(time.Date(2020, time.January, 5, 6, 3, 0, 1, time.Local).Unix()))
//
//				Expect(desc.Norm(time.Date(2020, time.January, 5, 6, 2, 0, 0, time.Local)).Unix()).
//					Should(Equal(time.Date(2020, time.January, 5, 6, 3, 0, 0, time.Local).Unix()))
//
//				Expect(desc.Norm(time.Date(2020, time.January, 5, 6, 3, 0, 0, time.Local)).Unix()).
//					Should(Equal(time.Date(2020, time.January, 5, 6, 3, 0, 0, time.Local).Unix()))
//
//				Expect(desc.Norm(time.Date(2020, time.January, 5, 6, 4, 0, 0, time.Local)).Unix()).
//					Should(Equal(time.Date(2020, time.January, 5, 6, 10, 0, 0, time.Local).Unix()))
//
//				Expect(desc.Norm(time.Date(2020, time.January, 5, 6, 10, 0, 0, time.Local)).Unix()).
//					Should(Equal(time.Date(2020, time.January, 5, 6, 10, 0, 0, time.Local).Unix()))
//
//				Expect(desc.Norm(time.Date(2020, time.January, 5, 6, 11, 0, 0, time.Local)).Unix()).
//					Should(Equal(time.Date(2020, time.January, 5, 6, 17, 0, 0, time.Local).Unix()))
//			})
//		})
//		Context("With hour configured", func() {
//			It("should be ok", func() {
//				desc := &CTDesc{hour: []*ctDescUnit{{begin: 3}}}
//				desc.init()
//				Expect(desc.Norm(time.Date(2020, time.January, 5, 0, 0, 0, 0, time.Local)).Unix()).
//					Should(Equal(time.Date(2020, time.January, 5, 3, 0, 0, 0, time.Local).Unix()))
//
//				Expect(desc.Norm(time.Date(2020, time.January, 5, 4, 0, 0, 0, time.Local)).Unix()).
//					Should(Equal(time.Date(2020, time.January, 6, 3, 0, 0, 0, time.Local).Unix()))
//			})
//		})
//		Context("With week configured", func() {
//			It("should be ok", func() {
//				desc := &CTDesc{week: []*ctDescUnit{{begin: 1, end: 5, step: 2}}}
//				desc.init()
//				Expect(desc.Norm(time.Date(2021, time.February, 9, 0, 0, 0, 0, time.Local)).Unix()).
//					Should(Equal(time.Date(2021, time.February, 10, 0, 0, 0, 0, time.Local).Unix()))
//
//				Expect(desc.Norm(time.Date(2021, time.February, 11, 0, 0, 0, 0, time.Local)).Unix()).
//					Should(Equal(time.Date(2021, time.February, 12, 0, 0, 0, 0, time.Local).Unix()))
//
//				Expect(desc.Norm(time.Date(2021, time.February, 11, 0, 0, 0, 0, time.Local)).Unix()).
//					Should(Equal(time.Date(2021, time.February, 12, 0, 0, 0, 0, time.Local).Unix()))
//
//				Expect(desc.Norm(time.Date(2021, time.February, 13, 0, 0, 0, 0, time.Local)).Unix()).
//					Should(Equal(time.Date(2021, time.February, 15, 0, 0, 0, 0, time.Local).Unix()))
//
//				Expect(desc.Norm(time.Date(2021, time.February, 27, 0, 0, 0, 0, time.Local)).Unix()).
//					Should(Equal(time.Date(2021, time.March, 1, 0, 0, 0, 0, time.Local).Unix()))
//			})
//		})
//		Context("With default configuration", func() {
//			It("should be ok", func() {
//				desc := &CTDesc{}
//				desc.init()
//
//				Expect(desc.Norm(time.Date(2000, time.December, 31, 23, 59, 59, 0, time.Local)).Unix()).
//					Should(Equal(time.Date(2000, time.December, 31, 23, 59, 59, 0, time.Local).Unix()))
//
//				Expect(desc.Norm(time.Date(2000, time.December, 31, 23, 59, 60, 0, time.Local)).Unix()).
//					Should(Equal(time.Date(2001, time.January, 1, 0, 0, 0, 0, time.Local).Unix()))
//			})
//		})
//		Context("With complex situation", func() {
//			It("should be ok", func() {
//				desc := &CTDesc{
//					month:  []*ctDescUnit{{begin: 2}},
//					day:    []*ctDescUnit{{begin: 29}},
//					hour:   []*ctDescUnit{{begin: 1}},
//					minute: []*ctDescUnit{{begin: 2}},
//					second: []*ctDescUnit{{begin: 3}},
//				}
//				desc.init()
//				Expect(desc.Norm(time.Date(2000, time.January, 1, 0, 0, 0, 0, time.Local)).Unix()).
//					Should(Equal(time.Date(2000, time.February, 29, 1, 2, 3, 0, time.Local).Unix()))
//
//				Expect(desc.Norm(time.Date(2000, time.February, 29, 0, 0, 0, 0, time.Local)).Unix()).
//					Should(Equal(time.Date(2000, time.February, 29, 1, 2, 3, 0, time.Local).Unix()))
//
//				Expect(desc.Norm(time.Date(2000, time.February, 29, 1, 2, 3, 0, time.Local)).Unix()).
//					Should(Equal(time.Date(2000, time.February, 29, 1, 2, 3, 0, time.Local).Unix()))
//
//				Expect(desc.Norm(time.Date(2000, time.February, 29, 1, 2, 4, 0, time.Local)).Unix()).
//					Should(Equal(time.Date(2004, time.February, 29, 1, 2, 3, 0, time.Local).Unix()))
//
//				Expect(desc.Norm(time.Date(2000, time.February, 30, 0, 0, 0, 0, time.Local)).Unix()).
//					Should(Equal(time.Date(2004, time.February, 29, 1, 2, 3, 0, time.Local).Unix()))
//			})
//		})
//	})
//})
