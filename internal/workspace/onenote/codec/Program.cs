using System.IO.Compression;
using System.Text;
using System.Text.Json.Nodes;
using OfficeIMO;
using OfficeIMO.Drawing;
using OfficeIMO.OneNote;

// The helper has no network or database access. Its only protocol is bounded bytes
// on stdin/stdout, with a JSON array of non-content conversion notes on stderr.
try
{
    if (args.Length != 1) throw new InvalidDataException("Choose a supported conversion.");
    byte[] input = ReadBounded(Console.OpenStandardInput(), 64 << 20);
    var codec = new OneNoteCodec();
    byte[] output = args[0] switch
    {
        "read-one" => codec.Read(input, false),
        "read-onepkg" => codec.Read(input, true),
        "write-one" => codec.Write(input, false),
        "write-onepkg" => codec.Write(input, true),
        _ => throw new InvalidDataException("Unsupported conversion.")
    };
    if (output.Length > 64 << 20) throw new InvalidDataException("The converted file exceeds 64 MiB.");
    Console.OpenStandardOutput().Write(output);
    Console.Error.WriteLine(new JsonArray(codec.Notes.Select(n => (JsonNode?)JsonValue.Create(n)).ToArray()).ToJsonString());
}
catch (Exception)
{
    // Do not echo decoder exceptions: they may contain private content or paths.
    Console.Error.WriteLine("The OneNote document is unsupported, encrypted, corrupt, or exceeds the conversion limits.");
    Environment.ExitCode = 1;
}
static byte[] ReadBounded(Stream stream, int limit)
{
    using var result = new MemoryStream(); var buffer = new byte[65536]; int count;
    while ((count = stream.Read(buffer)) > 0)
    {
        if (result.Length + count > limit) throw new InvalidDataException("Input limit exceeded.");
        result.Write(buffer, 0, count);
    }
    return result.ToArray();
}

internal sealed class OneNoteCodec
{
    const double Units = 48; // MS-ONE layout uses half-inch units; mindflow uses 96-DPI world units.
    readonly Dictionary<string, byte[]> files = new();
    readonly JsonArray assets = new();
    readonly JsonArray pages = new();
    public HashSet<string> Notes { get; } = new();
    int elements;
    long mediaBytes;
    public byte[] Read(byte[] input, bool package)
    {
        var limits = new OneNoteLoadOptions
        {
            MaxInputBytes = 64 << 20,
            MaxAssetBytes = 20 << 20,
            MaxTotalAssetBytes = 100 << 20,
            MaxObjects = 200000,
            MaxFileNodes = 200000,
            MaxPropertiesPerObject = 4096,
            MaxPropertySetDepth = 32,
            MaxPageGraphNodes = 10000,
            MaxInkPathValues = 40000,
            PreserveUnknownData = false
        };
        using var source = new MemoryStream(input, false);
        if (package)
        {
            var notebook = OneNotePackageReader.Read(source, "notebook.onepkg", new OneNoteNotebookReaderOptions
            {
                OneNoteOptions = limits,
                ContinueOnSectionError = false,
                IncludeRecycleBin = false,
                MaxPackageEntries = 1000,
                MaxPackageExpandedBytes = 128 << 20,
                MaxPackageEntryBytes = 64 << 20,
                MaxSectionGroupDepth = 16,
                MaxNotebookEntries = 1000
            });
            ReadSections(notebook.Sections, notebook.SectionGroups, "", 0);
        }
        else
        {
            using var section = OneNoteSection.Load(source, limits);
            ReadSection(section, "");
        }
        if (pages.Count == 0) throw new InvalidDataException("No active pages.");
        var manifest = new JsonObject { ["format"] = "mindflow", ["version"] = 1, ["kind"] = "workspace", ["name"] = "Imported OneNote notebook", ["pages"] = pages, ["assets"] = assets, ["templates"] = new JsonArray() };
        files["manifest.json"] = Encoding.UTF8.GetBytes(manifest.ToJsonString());
        Notes.Add("Page titles and order are preserved; sections are flattened into page-name prefixes. History, deleted content, sharing, tags, recordings, and attachments are not imported.");
        return Zip(files);
    }
    void ReadSections(IEnumerable<OneNoteSection> sections, IEnumerable<OneNoteSectionGroup> groups, string prefix, int depth)
    {
        if (depth > 16) throw new InvalidDataException();
        foreach (var section in sections) ReadSection(section, prefix);
        foreach (var group in groups) ReadSections(group.Sections, group.SectionGroups, prefix + group.Name + " / ", depth + 1);
    }
    void ReadSection(OneNoteSection section, string prefix)
    {
        foreach (var page in section.Pages)
        {
            if (page.IsDeleted || page.IsVersionHistoryPage || page.IsConflictPage) continue;
            if (pages.Count >= 100) throw new InvalidDataException("Too many pages.");
            var content = new JsonArray();
            foreach (var outline in page.Outlines) ReadElement(outline, content, 0, 0, 0);
            double flow = 0;
            foreach (var element in page.DirectContent) flow += ReadElement(element, content, 0, element.Layout?.Y is null ? flow : 0, 0);
            string name = prefix + section.Name + " / " + page.Title;
            if (name.Length > 190) name = name[..190];
            pages.Add((JsonNode)new JsonObject
            {
                ["id"] = "p" + pages.Count,
                ["name"] = string.IsNullOrWhiteSpace(name) ? "Page " + (pages.Count + 1) : name,
                ["document"] = new JsonObject { ["schemaVersion"] = 1, ["elements"] = content, ["settings"] = new JsonObject { ["background"] = "plain", ["backgroundColor"] = "#ffffff", ["gridSize"] = 20, ["snap"] = false } }
            });
        }
    }
    double ReadElement(OneNoteElement source, JsonArray content, double parentX, double parentY, int depth)
    {
        if (depth > 32 || ++elements > 100000 || content.Count >= 10000) throw new InvalidDataException();
        double x = parentX + (source.Layout?.X ?? 0) * Units;
        double y = parentY + (source.Layout?.Y ?? 0) * Units;
        double width = Math.Max(.1, (source.Layout?.Width ?? 8) * Units);
        double height = Math.Max(.1, (source.Layout?.Height ?? .5) * Units);
        switch (source)
        {
            case OneNoteOutline outline:
                double flow = 0;
                foreach (var child in outline.Children) flow += ReadElement(child, content, x, child.Layout?.Y is null ? y + flow : y, depth + 1);
                return Math.Max(height, flow);
            case OneNoteParagraph paragraph:
                string text = string.Concat(paragraph.Runs.Select(run => run.Text));
                if (text.Length > 100000) throw new InvalidDataException();
                var style = paragraph.Runs.FirstOrDefault()?.Style;
                var size = Math.Clamp(Math.Round((style?.FontSize ?? 12) * 4 / 3), 1, 2000);
                height = Math.Max(height, size * 1.3 * (1 + text.Count(c => c == '\n')));
                if (text.Length > 0)
                {
                    var element = Element("text", x, y, width, height);
                    element["text"] = text; element["fontSize"] = size;
                    element["color"] = Color(style?.ColorArgb ?? 0xff000000);
                    element["bold"] = style?.Bold ?? false; element["italic"] = style?.Italic ?? false;
                    element["align"] = paragraph.Style.Alignment switch { OneNoteParagraphAlignment.Center => "center", OneNoteParagraphAlignment.Right => "right", _ => "left" };
                    element["verticalAlign"] = "top";
                    content.Add((JsonNode)element);
                    Notes.Add("Rich text uses one style per element and mindflow font metrics; wrapping, tables, and list layout may differ.");
                }
                foreach (var child in paragraph.Children) height += ReadElement(child, content, x + 20, y + height, depth + 1);
                return height;
            case OneNoteImage image when image.Payload is not null:
                byte[] data = image.Payload.ToArray(20 << 20);
                string mime = ImageMime(data);
                string id = "a" + (assets.Count + 1);
                mediaBytes += data.Length;
                if (mediaBytes > 100 << 20) throw new InvalidDataException();
                assets.Add((JsonNode)new JsonObject { ["id"] = id, ["mime"] = mime }); files["assets/" + id + ".bin"] = data;
                width = Math.Max(.1, (image.WidthHalfInches ?? source.Layout?.Width ?? 6) * Units);
                height = Math.Max(.1, (image.HeightHalfInches ?? source.Layout?.Height ?? 4) * Units);
                var picture = Element("image", x, y, width, height); picture["assetId"] = id; content.Add((JsonNode)picture);
                return height;
            case OneNoteInk ink:
                foreach (var stroke in ink.Strokes) ReadStroke(stroke, content, x, y);
                return height;
            case OneNoteTable table:
                double rowY = y;
                foreach (var row in table.Rows)
                {
                    double colX = x, rowHeight = 0; int column = 0;
                    foreach (var cell in row.Cells)
                    {
                        double cellHeight = 0;
                        foreach (var child in cell.Content) cellHeight += ReadElement(child, content, colX, rowY + cellHeight, depth + 1);
                        rowHeight = Math.Max(rowHeight, cellHeight);
                        colX += (column < table.ColumnWidths.Count ? table.ColumnWidths[column] * Units : 160); column++;
                    }
                    rowY += Math.Max(24, rowHeight);
                }
                Notes.Add("Table contents become separate editable elements; table borders and cell formatting are not retained.");
                return rowY - y;
            default:
                Notes.Add("Unsupported math, embedded files, media, or opaque objects were omitted. Keep the source file."); return height;
        }
    }
    void ReadStroke(OfficeInkStroke source, JsonArray content, double x, double y)
    {
        if (source.Points.Count == 0) return;
        if (source.Points.Count > 20000 || content.Count >= 10000) throw new InvalidDataException();
        var points = source.Points.Select(p => source.Transform.HasValue ? p.Transform(source.Transform.Value) : p).ToArray();
        double left = points.Min(p => p.X) * Units + x, top = points.Min(p => p.Y) * Units + y;
        double width = Math.Max(.1, (points.Max(p => p.X) - points.Min(p => p.X)) * Units);
        double height = Math.Max(.1, (points.Max(p => p.Y) - points.Min(p => p.Y)) * Units);
        var stroke = Element("stroke", left, top, width, height); var coordinates = new JsonArray(); var pressures = new JsonArray();
        foreach (var p in points) { coordinates.Add((JsonNode?)JsonValue.Create((p.X * Units + x - left) / width)); coordinates.Add((JsonNode?)JsonValue.Create((p.Y * Units + y - top) / height)); pressures.Add((JsonNode?)JsonValue.Create(Math.Clamp(p.Pressure ?? 1, 0, 1))); }
        if (points.Length == 1) { coordinates.Add(coordinates[0]!.DeepClone()); coordinates.Add(coordinates[1]!.DeepClone()); pressures.Add(pressures[0]!.DeepClone()); }
        stroke["points"] = coordinates; stroke["color"] = $"#{source.Color.R:x2}{source.Color.G:x2}{source.Color.B:x2}";
        stroke["penWidth"] = Math.Clamp(source.Width * Units, .1, 100);
        if (source.IsHighlighter) stroke["highlight"] = true; else if (!source.IgnorePressure) stroke["pressures"] = pressures;
        content.Add((JsonNode)stroke);
        Notes.Add("Ink remains editable; pen-tip shape, opacity, and pressure rendering may differ.");
    }
    static JsonObject Element(string type, double left, double top, double width, double height) => new()
    {
        ["id"] = Guid.NewGuid().ToString("N"),
        ["type"] = type,
        ["x"] = left + width / 2,
        ["y"] = top + height / 2,
        ["width"] = width,
        ["height"] = height,
        ["rotation"] = 0
    };
    static string Color(uint argb) => (argb >> 24) == 255 ? $"#{argb & 0xffffff:x6}" : $"#{argb & 0xffffff:x6}{argb >> 24:x2}";
    static string ImageMime(byte[] b) => b.AsSpan().StartsWith(new byte[] { 137, 80, 78, 71 }) ? "image/png" : b.AsSpan().StartsWith(new byte[] { 255, 216 }) ? "image/jpeg" : b.Length > 12 && Encoding.ASCII.GetString(b, 8, 4) == "WEBP" ? "image/webp" : throw new InvalidDataException("Unsupported embedded image.");
    static byte[] Zip(Dictionary<string, byte[]> entries)
    {
        using var output = new MemoryStream();
        using (var zip = new ZipArchive(output, ZipArchiveMode.Create, true))
        {
            foreach (var (name, bytes) in entries) { using var stream = zip.CreateEntry(name).Open(); stream.Write(bytes); }
        }
        return output.ToArray();
    }
    public byte[] Write(byte[] input, bool package)
    {
        using var archive = new ZipArchive(new MemoryStream(input, false), ZipArchiveMode.Read);
        if (archive.Entries.Count > 2001) throw new InvalidDataException();
        var manifestEntry = archive.GetEntry("manifest.json") ?? throw new InvalidDataException();
        if (manifestEntry.Length > 32 << 20) throw new InvalidDataException();
        JsonNode manifest;
        using (var stream = manifestEntry.Open()) manifest = JsonNode.Parse(stream, documentOptions: new() { MaxDepth = 64 }) ?? throw new InvalidDataException();
        var sourcePages = manifest["pages"]!.AsArray();
        if (sourcePages.Count is < 1 or > 100) throw new InvalidDataException();
        using var section = OneNoteSection.Create();
        section.Name = manifest["name"]!.GetValue<string>();
        foreach (var sourcePage in sourcePages)
        {
            var page = new OneNotePage { Title = sourcePage!["name"]!.GetValue<string>() };
            var sourceElements = sourcePage["document"]!["elements"]!.AsArray();
            if (sourceElements.Count > 10000) throw new InvalidDataException();
            double offsetX = Math.Min(0, sourceElements.Select(e => Number(e!, "x") - Number(e!, "width") / 2).DefaultIfEmpty(0).Min()) - 32;
            double offsetY = Math.Min(0, sourceElements.Select(e => Number(e!, "y") - Number(e!, "height") / 2).DefaultIfEmpty(0).Min()) - 32;
            foreach (var element in sourceElements)
            {
                var e = element!;
                string type = e["type"]!.GetValue<string>();
                double width = Number(e, "width"), height = Number(e, "height");
                var layout = new OneNoteLayout { X = (Number(e, "x") - width / 2 - offsetX) / Units, Y = (Number(e, "y") - height / 2 - offsetY) / Units, Width = width / Units, Height = height / Units };
                if (type == "text")
                {
                    if (Math.Abs(Number(e, "rotation")) > .00001) throw new InvalidDataException("Rasterize rotated text before export.");
                    var paragraph = new OneNoteParagraph { Layout = layout };
                    var run = new OneNoteTextRun { Text = e["text"]?.GetValue<string>() ?? "" };
                    run.Style.FontSize = Number(e, "fontSize") * 3 / 4;
                    run.Style.ColorArgb = Argb(e["color"]?.GetValue<string>() ?? "#000000");
                    run.Style.Bold = e["bold"]?.GetValue<bool>() ?? false;
                    run.Style.Italic = e["italic"]?.GetValue<bool>() ?? false;
                    paragraph.Runs.Add(run);
                    paragraph.Style.Alignment = e["align"]?.GetValue<string>() switch { "center" => OneNoteParagraphAlignment.Center, "right" => OneNoteParagraphAlignment.Right, _ => OneNoteParagraphAlignment.Left };
                    page.DirectContent.Add(paragraph);
                    Notes.Add("Text remains editable; font metrics, vertical alignment, and wrapping can differ.");
                }
                else if (type == "image")
                {
                    if (Math.Abs(Number(e, "rotation")) > .00001 || e["crop"] is not null) throw new InvalidDataException("Bake image transforms before export.");
                    string id = e["assetId"]!.GetValue<string>();
                    var entry = archive.GetEntry("assets/" + id + ".bin") ?? throw new InvalidDataException();
                    if (entry.Length > 20 << 20) throw new InvalidDataException();
                    using var data = entry.Open(); using var buffer = new MemoryStream(); data.CopyTo(buffer);
                    page.DirectContent.Add(new OneNoteImage { Layout = layout, WidthHalfInches = width / Units, HeightHalfInches = height / Units, MediaType = ImageMime(buffer.ToArray()), FileName = "image.png", Payload = OneNoteBinaryPayload.FromBytes(buffer.ToArray()) });
                }
                else if (type == "stroke")
                {
                    var ink = new OneNoteInk { Layout = new OneNoteLayout { X = 0, Y = 0 } };
                    var stroke = new OfficeInkStroke { Width = Number(e, "penWidth") / Units, Height = Number(e, "penWidth") / Units, Color = OfficeColor.Parse(e["color"]!.GetValue<string>()), IsHighlighter = e["highlight"]?.GetValue<bool>() ?? false };
                    var points = e["points"]!.AsArray(); var pressures = e["pressures"]?.AsArray();
                    if (points.Count < 4 || points.Count > 40000 || points.Count % 2 != 0) throw new InvalidDataException();
                    var angle = Number(e, "rotation");
                    for (int i = 0; i < points.Count; i += 2)
                    {
                        double px = (points[i]!.GetValue<double>() - .5) * width, py = (points[i + 1]!.GetValue<double>() - .5) * height;
                        double x = Number(e, "x") + px * Math.Cos(angle) - py * Math.Sin(angle) - offsetX;
                        double y = Number(e, "y") + px * Math.Sin(angle) + py * Math.Cos(angle) - offsetY;
                        stroke.AddPoint(x / Units, y / Units, pressures is null ? null : pressures[i / 2]!.GetValue<double>());
                    }
                    stroke.IgnorePressure = pressures is null; ink.Ink.Add(stroke); page.DirectContent.Add(ink);
                    Notes.Add("Ink remains editable; pen-tip shape, highlight blending, and pressure rendering can differ.");
                }
                else throw new InvalidDataException("Rasterize unsupported elements before export.");
            }
            section.Pages.Add(page);
        }
        var options = new OneNoteSaveOptions { MaxOutputBytes = 64 << 20, MaxPackageEntries = 1000, ValidateRoundTrip = true };
        Notes.Add("Canvas backgrounds/grids, sharing, comments, templates, group locks, and element links are not exported. Pages are shifted into positive OneNote coordinates with a margin.");
        if (!package) return section.ToBytes(options);
        var notebook = new OneNoteNotebook { Name = section.Name }; notebook.Sections.Add(section);
        return OneNotePackageWriter.Write(notebook, options);
    }
    static double Number(JsonNode node, string name)
    {
        var number = node[name]?.GetValue<double>() ?? 0;
        if (!double.IsFinite(number) || Math.Abs(number) > 1000000000) throw new InvalidDataException();
        return number;
    }
    static uint Argb(string color)
    {
        var rgba = OfficeColor.Parse(color);
        return ((uint)rgba.A << 24) | ((uint)rgba.R << 16) | ((uint)rgba.G << 8) | rgba.B;
    }
}
