package site

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runSky runs a script under node with zones.js, art.js and sky.js loaded
// into a bare global object (no document), so the sky's arithmetic is
// tested without a browser.
func runSky(t *testing.T, body string) string {
	t.Helper()
	return runSkyEnv(t, nil, body)
}

// runSkyEnv is runSky with extra environment variables, such as TZ.
func runSkyEnv(t *testing.T, env []string, body string) string {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed")
	}
	var src strings.Builder
	for _, f := range []string{"zones.js", "art.js", "sky.js"} {
		b, err := os.ReadFile(filepath.Join("assets", f))
		if err != nil {
			t.Fatal(err)
		}
		src.Write(b)
		src.WriteString("\n")
	}
	script := "var window = {};\n" + src.String() + "var k = window.ccbSky;\n" +
		"function hm(m) { m = Math.round(((m % 1440) + 1440) % 1440); return String(Math.floor(m / 60)).padStart(2, '0') + ':' + String(m % 60).padStart(2, '0'); }\n" + body
	// The script goes into a file, not onto the command line: with the time
	// zone table it is longer than Windows allows a command line to be.
	file := filepath.Join(t.TempDir(), "sky-test.js")
	if err := os.WriteFile(file, []byte(script), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(node, file)
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node failed: %v\n%s", err, out)
	}
	return strings.TrimSpace(string(out))
}

func TestSkySunTimes(t *testing.T) {
	got := runSky(t, `
var hel = k.placeFor("Europe/Helsinki"), syd = k.placeFor("Australia/Sydney");
function show(s) { return hm(s.rise) + "-" + hm(s.set) + " " + Math.round(s.maxAlt) + (s.south ? " S" : ""); }
console.log(show(k.sunDay(172, hel, 180)));
console.log(show(k.sunDay(355, hel, 120)));
console.log(show(k.sunDay(355, syd, 660)));
console.log(show(k.sunDay(100, null, 0)));`)
	want := "03:54-22:50 53\n09:25-15:13 6\n05:42-20:06 80 S\n06:00-18:00 70"
	if got != want {
		t.Fatalf("sun times:\n%s\nwant:\n%s", got, want)
	}
}

func TestSkyPolarDays(t *testing.T) {
	got := runSky(t, `
var tro = {lat: 69.65, lon: 18.96};
var june = k.sunDay(172, tro, 120), dec = k.sunDay(355, tro, 60);
console.log(june.kind + "|" + dec.kind);
var g = {W: 375, H: 300, rooH: 160, size: 36};
[["light", june], ["dark", june], ["light", dec], ["dark", dec]].forEach(function (c) {
  var w = k.skyWindow(c[0], c[1]);
  for (var m = 0; m < 1440; m += 37) {
    var p = k.position(m, w), a = k.arcPoint(p.t, c[0], c[1], g), line = k.captionText(m, c[0], c[1], p, 10, c[1] === june ? 6 : 12, 21);
    if (!isFinite(a.x) || !isFinite(a.y) || a.y > g.H - 0.1 * g.size - g.size / 2 + 1) { console.log("bad " + c[0] + " " + m); }
    if (line.indexOf("NaN") >= 0 || line.indexOf("undefined") >= 0) { console.log("bad line " + line); }
  }
});
console.log(k.captionText(600, "light", dec, k.position(600, k.skyWindow("light", dec)), 10, 12, 21));
console.log(k.captionText(600, "dark", june, k.position(600, k.skyWindow("dark", june)), 10, 6, 21));
console.log(k.captionText(30, "light", june, k.position(30, k.skyWindow("light", june)), 10, 6, 21));`)
	want := "midnight sun|polar night\n10:00 \u00b7 polar night\n10:00 \u00b7 midnight sun\n00:30 \u00b7 midnight sun"
	if got != want {
		t.Fatalf("polar days:\n%s\nwant:\n%s", got, want)
	}
}

func TestSkyPlaceForZones(t *testing.T) {
	got := runSky(t, `
["Europe/Helsinki", "Asia/Calcutta", "Europe/Kiev", "UTC", "Etc/GMT+3", "", "Nowhere/Else"].forEach(function (z) {
  var p = k.placeFor(z); console.log(z + "=" + (p ? p.lat.toFixed(1) + "," + p.lon.toFixed(1) : "none"));
});`)
	want := "Europe/Helsinki=60.2,25.0\nAsia/Calcutta=22.5,88.4\nEurope/Kiev=50.4,30.5\nUTC=none\nEtc/GMT+3=none\n=none\nNowhere/Else=none"
	if got != want {
		t.Fatalf("places:\n%s\nwant:\n%s", got, want)
	}
}

func TestSkyEveryBrowserZoneResolves(t *testing.T) {
	got := runSky(t, `
var misses = Intl.supportedValuesOf("timeZone").filter(function (z) {
  return z !== "UTC" && z.indexOf("Etc/") !== 0 && !k.placeFor(z);
});
console.log(misses.join(",") + "|" + Intl.supportedValuesOf("timeZone").length);`)
	if !strings.HasPrefix(got, "|") {
		t.Fatalf("time zones the browser reports but the table does not know: %s", strings.SplitN(got, "|", 2)[0])
	}
}

func TestSkyWaitingAndPeeking(t *testing.T) {
	got := runSky(t, `
var s = k.sunDay(100, null, 0), g = {W: 1000, H: 400, rooH: 208, size: 45};
var lightNight = k.position(0, k.skyWindow("light", s)), darkNoon = k.position(720, k.skyWindow("dark", s));
console.log(lightNight.t + " " + lightNight.untilRise + " " + darkNoon.t + " " + darkNoon.untilRise);
var wait = k.arcPoint(null, "light", s, g), rise = k.arcPoint(0, "light", s, g), noon = k.arcPoint(0.5, "light", s, g);
console.log((wait.x === rise.x && wait.y === rise.y) + " " + (wait.x < 500) + " " + (400 - wait.y >= 0.6 * 45 - 1) + " " + (noon.y < 400 - 208));
var south = k.sunDay(355, k.placeFor("Australia/Sydney"), 660);
console.log(k.arcPoint(0.1, "light", south, g).x > 500);`)
	want := "null 360 null 360\ntrue true true true\ntrue"
	if got != want {
		t.Fatalf("waiting and peeking:\n%s\nwant:\n%s", got, want)
	}
}

func TestSkyAcrossMidnight(t *testing.T) {
	got := runSky(t, `
var s = k.sunDay(100, null, 0), wl = k.skyWindow("light", s), wd = k.skyWindow("dark", s);
console.log(k.position(1439, wl).untilRise + " " + k.position(1, wl).untilRise);
var a = k.position(1439, wd).t, b = k.position(0, wd).t, c = k.position(1, wd).t;
console.log((a < b && b < c) + " " + (k.position(385, wd).t !== null) + " " + (k.position(386, wd).t === null));`)
	want := "361 359\ntrue true true"
	if got != want {
		t.Fatalf("across midnight:\n%s\nwant:\n%s", got, want)
	}
}

func TestSkyMoon(t *testing.T) {
	got := runSky(t, `
console.log(k.moonAge(Date.UTC(2000, 0, 6, 18, 14)).toFixed(2));
[0.2, 4, 7.4, 11, 14.8, 18, 22.1, 27].forEach(function (a) { console.log(a + " " + k.phaseName(a) + " " + k.litPercent(a)); });`)
	want := "0.00\n0.2 new moon 0\n4 waxing crescent 17\n7.4 first quarter 50\n11 waxing gibbous 85\n14.8 full moon 100\n18 waning gibbous 89\n22.1 last quarter 51\n27 waning crescent 7"
	if got != want {
		t.Fatalf("moon:\n%s\nwant:\n%s", got, want)
	}
}

func TestSkyCaptions(t *testing.T) {
	got := runSky(t, `
var s = k.sunDay(100, null, 0);
function line(m, theme, age, mon, dom) { return k.captionText(m, theme, s, k.position(m, k.skyWindow(theme, s)), age, mon || 4, dom || 10); }
console.log(line(720, "light", 4));
console.log(line(0, "light", 4));
console.log(line(1260, "dark", 4));
console.log(line(1260, "dark", 14.8));
console.log(line(1260, "dark", 0.2));
console.log(line(720, "dark", 4));
console.log(line(720, "light", 4, 9, 22));
console.log(line(720, "light", 4, 3, 20));
console.log(line(720, "light", 4, 3, 19));
var hel = k.placeFor("Europe/Helsinki"), sd = k.sunDay(172, hel, 180);
console.log(k.captionText(780, "light", sd, k.position(780, k.skyWindow("light", sd)), 4, 6, 21));
var sy = k.sunDay(172, k.placeFor("Australia/Sydney"), 600);
console.log(k.captionText(720, "light", sy, k.position(720, k.skyWindow("light", sy)), 4, 6, 21));
var sm = k.sunDay(80, k.placeFor("Australia/Sydney"), 660);
console.log(k.captionText(720, "light", sm, k.position(720, k.skyWindow("light", sm)), 4, 3, 20));
var hm = k.sunDay(80, hel, 120);
console.log(k.captionText(720, "light", hm, k.position(720, k.skyWindow("light", hm)), 4, 3, 20));`)
	want := strings.Join([]string{
		"12:00 \u00b7 sunset 18:00",
		"00:00 \u00b7 sunrise in 6h 00m",
		"21:00 \u00b7 waxing crescent, 17%",
		"21:00 \u00b7 full moon",
		"21:00 \u00b7 new moon",
		"12:00 \u00b7 moonrise in 6h 00m",
		"12:00 \u00b7 autumn equinox",
		"12:00 \u00b7 spring equinox",
		"12:00 \u00b7 sunset 18:00",
		"13:00 \u00b7 longest day",
		"12:00 \u00b7 shortest day",
		"12:00 \u00b7 autumn equinox",
		"12:00 \u00b7 spring equinox",
	}, "\n")
	if got != want {
		t.Fatalf("captions:\n%s\nwant:\n%s", got, want)
	}
}

func TestSkyLineAvoidsBoxes(t *testing.T) {
	got := runSky(t, `
function measure(s) { var lines = s.split("\n"); return {w: Math.max.apply(null, lines.map(function (l) { return l.length; })) * 6.3, h: lines.length * 14}; }
var sky = {W: 375, H: 300}, roo = [[107, 140, 267, 300]];
function overlaps(p) { var m = measure(p.text); return roo.some(function (b) { return p.x < b[2] && p.x + m.w > b[0] && p.y < b[3] && p.y + m.h > b[1]; }); }
function inside(p) { var m = measure(p.text); return p.x >= 0 && p.y >= 0 && p.x + m.w <= sky.W && p.y + m.h <= sky.H; }
var text = "16:00 \u00b7 sunrise in 17h 22m", bad = 0;
for (var x = 0; x <= 375 - 36; x += 9) for (var y = 20; y <= 300 - 33; y += 9) {
  var p = k.placeCaption(text, {x: x, y: y, size: 36}, sky, roo, measure);
  if (overlaps(p) || !inside(p)) bad++;
}
console.log(bad);
var open = k.placeCaption("12:00 \u00b7 sunset 18:00", {x: 400, y: 50, size: 45}, {W: 1000, H: 400}, [], measure);
console.log(open.text.indexOf("\n") < 0 && open.y > 50 + 45);
var low = k.placeCaption("06:10 \u00b7 sunset 18:00", {x: 10, y: 400 - 41, size: 45}, {W: 1000, H: 400}, [], measure);
console.log(low.y < 400 - 41);`)
	if got != "0\ntrue\ntrue" {
		t.Fatalf("line placement:\n%s", got)
	}
}

func TestArtFootprintCoversEveryPose(t *testing.T) {
	got := runSky(t, `
var f = window.ccbArt.footprint(), seen = {};
f.forEach(function (c) { seen[c[0] + "," + c[1]] = true; });
console.log(f.length > 100, !!seen["14,1"], !!seen["13,9"], !!seen["0,11"]);`)
	if got != "true true true true" {
		t.Fatalf("footprint: %s", got)
	}
}

// TestSkyDayAcrossClockChange checks the day of the year and the UTC offset
// the sun is reckoned with on the night the clocks go forward (8 March 2026
// in New York): the offset is the one at noon of that day, so the hour
// before the change already has the day's sunrise and sunset. Just after
// midnight on the day after a change, the day is still counted by the
// calendar, not by the hours since the new year, which are one short in
// spring and one over in autumn.
func TestSkyDayAcrossClockChange(t *testing.T) {
	got := runSkyEnv(t, []string{"TZ=America/New_York"}, `
var a = new Date(2026, 2, 8, 0, 30), b = new Date(2026, 2, 8, 3, 30);
console.log([k.localDay(a), k.localDay(b), k.noonOffset(a), k.noonOffset(b)].join(" "));
console.log([k.localDay(new Date(2026, 2, 9, 0, 30)), k.localDay(new Date(2026, 10, 1, 0, 30))].join(" "));
`)
	if want := "67 67 -240 -240\n68 305"; got != want {
		t.Errorf("day and offset across the clock change = %q, want %q", got, want)
	}
}
