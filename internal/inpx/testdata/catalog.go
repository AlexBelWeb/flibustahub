package testdata

import (
	"bytes"
	"time"
)

var stamp = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

const (
	ArchiveLow     = "d.fb2-000001-000100.zip"
	ArchiveHigh    = "f.fb2-000200-000300.zip"
	ArchiveMissing = "z.fb2-000400-000500.zip"
	InpLow         = "d.fb2-000001-000100.inp"
	InpHigh        = "f.fb2-000200-000300.inp"
	InpMissing     = "z.fb2-000400-000500.inp"
	DumpName       = "flibusta_fb2_local.inpx"
)

// Gromov is the corrected example record.
func Gromov() [14]string {
	return field(
		"Громов,Александр,Николаевич:",
		"sf_social:",
		"Первый из могикан",
		"Мир матриархата",
		"2",
		"110119",
		"1230745",
		"110119",
		"0",
		"fb2",
		"2008-07-05",
		"ru",
		"3",
		"",
	)
}

func CoauthorsOrderA() [14]string {
	return gromovOriginalOrderWithPetrov()
}

func CoauthorsOrderB() [14]string {
	return gromovSwappedAuthors()
}

func ManyAuthors() [14]string { return manyAuthors() }

func NoAuthors() [14]string { return noAuthors() }

func Deleted() [14]string { return deleted() }

func NoLibID() [14]string { return noLibID() }

func SernoRange() [14]string { return sernoRange() }

func SernoQuestion() [14]string { return sernoQuestion() }

func SharedLibidLow() [14]string { return sharedLibidLow() }

func SharedLibidHigh() [14]string { return sharedLibidHigh() }

func DelOne() [14]string { return delOne() }

func DelZero() [14]string { return delZero() }

func UnnamedGenre() [14]string { return unnamedGenre() }

func YoTitle() [14]string { return yoTitle() }

func NoLFTail() [14]string { return noLFTail() }

func MissingArchiveBook() [14]string { return missingArchiveBook() }

func gromovSwappedAuthors() [14]string {
	f := Gromov()
	f[0] = "Петров,Пётр,Петрович:Громов,Александр,Николаевич:"
	f[2] = "Первый из могикан"
	f[5] = "110120"
	f[7] = "110120"
	return f
}

func gromovOriginalOrderWithPetrov() [14]string {
	f := Gromov()
	f[0] = "Громов,Александр,Николаевич:Петров,Пётр,Петрович:"
	f[5] = "110121"
	f[7] = "110121"
	return f
}

func manyAuthors() [14]string {
	return field(
		nAuthors(30),
		"sf:",
		"Сборник тридцати авторов",
		"",
		"",
		"300001",
		"100",
		"300001",
		"0",
		"fb2",
		"2020-01-01",
		"ru",
		"",
		"",
	)
}

func noAuthors() [14]string {
	return field(
		"",
		"prose_contemporary:",
		"Книга без авторов",
		"",
		"",
		"400001",
		"50",
		"400001",
		"0",
		"fb2",
		"2019-02-02",
		"ru",
		"",
		"",
	)
}

func deleted() [14]string {
	f := Gromov()
	f[2] = "Удалённая книга"
	f[5] = "500001"
	f[7] = "500001"
	f[8] = "1"
	return f
}

func noLibID() [14]string {
	f := Gromov()
	f[2] = "Без идентификатора"
	f[5] = "600001"
	f[7] = ""
	return f
}

func sernoRange() [14]string {
	return field(
		"Иванов,Иван,Иванович:",
		"sf_action:",
		"Том с диапазоном",
		"Серия Мусор",
		"1-2",
		"700001",
		"10",
		"700001",
		"0",
		"fb2",
		"2018-03-03",
		"ru",
		"",
		"",
	)
}

func sernoQuestion() [14]string {
	return field(
		"Иванов,Иван,Иванович:",
		"sf_action:",
		"Том с вопросом",
		"Серия Мусор",
		"?",
		"700002",
		"11",
		"700002",
		"0",
		"fb2",
		"2018-03-04",
		"ru",
		"",
		"",
	)
}

func missingArchiveBook() [14]string {
	return field(
		"Сидоров,Сидор,:",
		"sf:",
		"Книга из отсутствующего архива",
		"",
		"",
		"800001",
		"20",
		"800001",
		"0",
		"fb2",
		"2017-04-04",
		"en",
		"",
		"",
	)
}

func sharedLibidLow() [14]string {
	return field(
		"Козлов,Козёл,:",
		"sf:",
		"Один libid в старом архиве",
		"",
		"",
		"900001",
		"30",
		"900001",
		"0",
		"fb2",
		"2016-05-05",
		"ru",
		"",
		"",
	)
}

func sharedLibidHigh() [14]string {
	f := sharedLibidLow()
	f[2] = "Один libid в новом архиве"
	f[5] = "900001"
	return f
}

func delOne() [14]string {
	return field(
		"Новиков,Ник,:",
		"sf:",
		"Переход удаления",
		"",
		"",
		"910001",
		"40",
		"910001",
		"1",
		"fb2",
		"2015-06-06",
		"ru",
		"",
		"",
	)
}

func delZero() [14]string {
	f := delOne()
	f[8] = "0"
	f[2] = "Переход удаления живая"
	return f
}

func unnamedGenre() [14]string {
	return field(
		"Жанров,Без,:",
		"totally_unknown_genre:",
		"Книга с неизвестным жанром",
		"",
		"",
		"920001",
		"15",
		"920001",
		"0",
		"fb2",
		"2014-07-07",
		"ru",
		"",
		"",
	)
}

func biographyLabel() [14]string {
	f := Gromov()
	f[1] = "Биографии и мемуары:"
	f[2] = "Книга с названием жанра"
	f[5] = "950001"
	f[7] = "950001"
	return f
}

func espionageLabel() [14]string {
	f := Gromov()
	f[1] = "Шпионский Детектив:"
	f[2] = "Книга со шпионским жанром"
	f[5] = "950002"
	f[7] = "950002"
	return f
}

func ambiguousGenreLabel() [14]string {
	f := Gromov()
	f[1] = "Дамский детективный роман:"
	f[2] = "Книга с неоднозначным жанром"
	f[5] = "950003"
	f[7] = "950003"
	return f
}

func yoTitle() [14]string {
	return field(
		"Ёлкин,Ёж,:",
		"sf:",
		"Ёлка",
		"",
		"",
		"930001",
		"16",
		"930001",
		"0",
		"fb2",
		"2013-08-08",
		"ru",
		"",
		"",
	)
}

func noLFTail() [14]string {
	return field(
		"Хвостов,Безперевода,:",
		"sf:",
		"Последняя без LF",
		"",
		"",
		"940001",
		"17",
		"940001",
		"0",
		"fb2",
		"2012-09-09",
		"ru",
		"",
		"",
	)
}

func versionInfo(bom bool, value string) []byte {
	var b bytes.Buffer
	if bom {
		b.Write([]byte{0xEF, 0xBB, 0xBF})
	}
	b.WriteString(value)
	return b.Bytes()
}

func versionInfoCP1251() []byte {
	return cp1251("20260901\nКириллический хвост")
}

func collectionInfo() []byte {
	return []byte("Flibusta local\nflibusta_20260901\n")
}

func collectionInfoUTF8() []byte {
	return []byte("Флибуста\nfrom_collection_utf8_20250101\n")
}

func collectionInfoCP1251() []byte {
	return cp1251("Флибуста\nfrom_collection_cp1251_20250101\n")
}

// CatalogINPX is the main synthetic dump. .inp members are added in reverse
// name order so the parser must sort by filename, not zip order.
func CatalogINPX() []byte {
	low := bytes.Join([][]byte{
		Record(Gromov(), true),
		Record(gromovOriginalOrderWithPetrov(), true),
		Record(gromovSwappedAuthors(), true),
		Record(manyAuthors(), true),
		Record(noAuthors(), true),
		Record(deleted(), true),
		Record(noLibID(), true),
		BrokenRecord("битая", "запись", "мало"),
		Record(sernoRange(), true),
		Record(sernoQuestion(), true),
		Record(sharedLibidLow(), true),
		Record(delOne(), true),
		Record(unnamedGenre(), true),
		Record(yoTitle(), true),
		Record(noLFTail(), false), // last record has no terminating LF
	}, nil)

	high := bytes.Join([][]byte{
		Record(sharedLibidHigh(), true),
		Record(delZero(), true),
	}, nil)

	missing := Record(missingArchiveBook(), true)

	return writeZip([]zipEntry{
		{name: InpMissing, body: missing},
		{name: InpHigh, body: high},
		{name: InpLow, body: low},
		{name: "version.info", body: versionInfo(true, "20260901")},
		{name: "collection.info", body: collectionInfo()},
		{name: "structure.info", body: []byte("ignore-me")},
	}, stamp)
}

// CollectionOnlyINPX has no version.info; version is the second line of collection.info.
func CollectionOnlyINPX() []byte {
	return writeZip([]zipEntry{
		{name: InpLow, body: Record(Gromov(), true)},
		{name: "collection.info", body: []byte("Title line\nfrom_collection_20250101\n")},
	}, stamp)
}

// UTF8CollectionINPX has UTF-8 collection.info with a Cyrillic first line.
func UTF8CollectionINPX() []byte {
	return writeZip([]zipEntry{
		{name: InpLow, body: RecordUTF8(Gromov(), true)},
		{name: "collection.info", body: collectionInfoUTF8()},
	}, stamp)
}

// CP1251CollectionINPX has CP1251 collection.info with a Cyrillic first line.
func CP1251CollectionINPX() []byte {
	return writeZip([]zipEntry{
		{name: InpLow, body: Record(Gromov(), true)},
		{name: "collection.info", body: collectionInfoCP1251()},
	}, stamp)
}

// CP1251VersionINPX has version.info that is not valid UTF-8.
func CP1251VersionINPX() []byte {
	return writeZip([]zipEntry{
		{name: InpLow, body: Record(Gromov(), true)},
		{name: "version.info", body: versionInfoCP1251()},
	}, stamp)
}

// MixedEncodingINPX has one UTF-8 .inp and one CP1251 .inp.
func MixedEncodingINPX() []byte {
	return writeZip([]zipEntry{
		{name: InpLow, body: RecordUTF8(Gromov(), true)},
		{name: InpHigh, body: Record(YoTitle(), true)},
		{name: "version.info", body: versionInfo(false, "20260901")},
		{name: "collection.info", body: collectionInfoCP1251()},
	}, stamp)
}

// GenreLabelsINPX has GENRE values that are dictionary names, not codes.
func GenreLabelsINPX() []byte {
	body := bytes.Join([][]byte{
		Record(biographyLabel(), true),
		Record(espionageLabel(), true),
		Record(ambiguousGenreLabel(), true),
	}, nil)
	return writeZip([]zipEntry{
		{name: InpLow, body: body},
		{name: "version.info", body: versionInfo(false, "20260901")},
	}, stamp)
}

// StaleINPX is an older dump of the same name, for non-recursive autosearch tests.
func StaleINPX() []byte {
	return writeZip([]zipEntry{
		{name: InpLow, body: Record(Gromov(), true)},
		{name: "version.info", body: versionInfo(false, "20251202")},
	}, time.Date(2025, 12, 2, 0, 0, 0, 0, time.UTC))
}

func EmptyArchiveZip() []byte { return emptyZip() }
