/* eslint-disable */
// Mock data for the P2P Market Terminal prototype.
// Mirrors the real types in frontend/src/types/marketData.ts so the prototype
// can be wired into the codebase without reshaping.

const NOW = Date.now();
const days = (n) => new Date(NOW - n * 86400000);
const mins = (n) => new Date(NOW - n * 60000);

// ----- TICKER (top strip) -----
const TICKER = [
  { sym: "AAPL",  px: 234.18, chg:  +1.42 },
  { sym: "MSFT",  px: 472.06, chg:  +0.31 },
  { sym: "NVDA",  px: 142.83, chg:  -2.18 },
  { sym: "TSLA",  px: 268.41, chg:  +3.07 },
  { sym: "AMZN",  px: 211.55, chg:  +0.62 },
  { sym: "GOOGL", px: 175.92, chg:  -0.44 },
  { sym: "META",  px: 612.18, chg:  +1.85 },
  { sym: "AMD",   px:  98.71, chg:  -3.42 },
  { sym: "JPM",   px: 248.66, chg:  +0.18 },
  { sym: "BRK.B", px: 481.30, chg:  +0.07 },
  { sym: "BTC",   px: 96412.0, chg: +2.64 },
  { sym: "ETH",   px: 3284.5,  chg: -0.81 },
  { sym: "GLD",   px: 248.91, chg:  +0.24 },
  { sym: "TLT",   px:  88.42, chg:  -0.55 },
  { sym: "SPY",   px: 582.14, chg:  +0.48 },
  { sym: "QQQ",   px: 498.27, chg:  +0.71 },
];

// ----- KPIs (dashboard) -----
const KPIS = {
  peers:     { val: 47,  delta: +6,    sub: "12 verified · 3 stale" },
  transfers: { val:  9,  delta: -2,    sub: "2 inbound · 7 outbound" },
  datasets:  { val: 1284, delta: +112, sub: "12.8 GB local" },
  symbols:   { val: 3142, delta: +28,  sub: "EOD · DIV · INS · SPL" },
  uptime:    { val: 99.94, delta: +0.02, sub: "blocks: 0 · errors: 1" },
  hitrate:   { val: 86.2, delta: +1.4, sub: "7d rolling" },
};

// ----- EOD MARKET DATA (dense table) -----
const EOD = [
  ["AAPL", "234.18", "+1.42",  "+0.61",  "234.65", "232.07", "232.84", "232.76", "58.4M", "EOD"],
  ["MSFT", "472.06", "+0.31",  "+0.07",  "473.52", "470.81", "471.98", "471.75", "21.6M", "EOD"],
  ["NVDA", "142.83", "-2.18",  "-1.50",  "146.12", "142.21", "145.01", "145.01", "212.4M","EOD"],
  ["TSLA", "268.41", "+3.07",  "+1.16",  "269.88", "263.62", "265.34", "265.34", "94.7M", "EOD"],
  ["AMZN", "211.55", "+0.62",  "+0.29",  "212.46", "209.88", "210.93", "210.93", "38.2M", "EOD"],
  ["GOOGL","175.92", "-0.44",  "-0.25",  "176.71", "174.92", "176.36", "176.36", "27.1M", "EOD"],
  ["META", "612.18", "+1.85",  "+0.30",  "613.41", "606.55", "610.33", "610.33", "12.8M", "EOD"],
  ["AMD",  "98.71",  "-3.42",  "-3.35",  "102.61", "98.40",  "102.13", "102.13","78.6M", "EOD"],
  ["JPM",  "248.66", "+0.18",  "+0.07",  "249.21", "246.92", "248.48", "248.48", "9.4M",  "EOD"],
  ["BRK.B","481.30", "+0.07",  "+0.01",  "482.05", "479.66", "481.23", "481.23", "3.1M",  "EOD"],
  ["XOM",  "120.44", "-0.92",  "-0.76",  "121.66", "120.01", "121.36", "121.36", "14.6M", "EOD"],
  ["CVX",  "162.18", "+0.41",  "+0.25",  "162.82", "160.71", "161.77", "161.77", "6.8M",  "EOD"],
  ["WMT",  " 92.74", "+0.55",  "+0.60",  " 93.01", " 92.10", " 92.19", " 92.19","18.7M", "EOD"],
  ["KO",   " 71.06", "+0.08",  "+0.11",  " 71.22", " 70.84", " 70.98", " 70.98","11.2M", "EOD"],
  ["PEP",  "151.83", "-0.30",  "-0.20",  "152.41", "151.22", "152.13", "152.13", "4.6M",  "EOD"],
].map(([sym,c,chg,pct,h,l,o,prev,vol,t]) => ({
  symbol: sym.trim(), close: c, chg, pct, high: h, low: l, open: o, prevClose: prev, volume: vol, type: t,
}));

// ----- DIVIDENDS -----
const DIVIDENDS = [
  { sym:"AAPL",  date:"2026-05-12", amount: 0.25, type:"Cash", currency:"USD", price: 234.18, src:"peer-7a3f" },
  { sym:"MSFT",  date:"2026-05-15", amount: 0.83, type:"Cash", currency:"USD", price: 472.06, src:"peer-c812" },
  { sym:"JPM",   date:"2026-05-09", amount: 1.25, type:"Cash", currency:"USD", price: 248.66, src:"peer-2f1a" },
  { sym:"KO",    date:"2026-05-01", amount: 0.48, type:"Cash", currency:"USD", price:  71.06, src:"peer-7a3f" },
  { sym:"XOM",   date:"2026-04-28", amount: 0.99, type:"Cash", currency:"USD", price: 120.44, src:"peer-c812" },
  { sym:"CVX",   date:"2026-04-24", amount: 1.71, type:"Cash", currency:"USD", price: 162.18, src:"peer-9d4e" },
  { sym:"BRK.B", date:"2026-04-15", amount: 0.00, type:"None", currency:"USD", price: 481.30, src:"local" },
  { sym:"PEP",   date:"2026-04-12", amount: 1.42, type:"Cash", currency:"USD", price: 151.83, src:"peer-2f1a" },
];

// ----- INSIDER TRADES -----
const INSIDER = [
  { sym:"NVDA",  date:"2026-05-21", name:"Huang, Jen-Hsun",  pos:"CEO",        ttype:"SELL", shares:120000, price: 144.21, value: 17305200, form:"Form 4" },
  { sym:"TSLA",  date:"2026-05-20", name:"Musk, Elon R.",     pos:"CEO",        ttype:"BUY",  shares: 50000, price: 266.30, value: 13315000, form:"Form 4" },
  { sym:"AAPL",  date:"2026-05-18", name:"Cook, Timothy D.",  pos:"CEO",        ttype:"SELL", shares: 25000, price: 232.91, value:  5822750, form:"Form 4" },
  { sym:"META",  date:"2026-05-16", name:"Sandberg, S.",      pos:"Director",   ttype:"SELL", shares:  4500, price: 610.84, value:  2748780, form:"Form 4" },
  { sym:"MSFT",  date:"2026-05-13", name:"Nadella, Satya",    pos:"CEO",        ttype:"SELL", shares:  9000, price: 471.40, value:  4242600, form:"Form 4" },
  { sym:"AMD",   date:"2026-05-08", name:"Su, Lisa T.",       pos:"CEO",        ttype:"BUY",  shares:  8000, price: 101.04, value:   808320, form:"Form 4" },
  { sym:"GOOGL", date:"2026-05-03", name:"Pichai, Sundar",    pos:"CEO",        ttype:"SELL", shares: 12000, price: 176.21, value:  2114520, form:"Form 4" },
];

// ----- SPLITS -----
const SPLITS = [
  { sym:"NVDA",  date:"2024-06-10", ratio:"10:1",  prev: 1208.88, post: 120.88, src:"peer-9d4e" },
  { sym:"TSLA",  date:"2022-08-25", ratio:"3:1",   prev:  890.42, post: 296.81, src:"peer-7a3f" },
  { sym:"AAPL",  date:"2020-08-31", ratio:"4:1",   prev:  499.23, post: 124.81, src:"local" },
  { sym:"GOOGL", date:"2022-07-18", ratio:"20:1",  prev: 2255.20, post: 112.76, src:"peer-c812" },
  { sym:"AMZN",  date:"2022-06-06", ratio:"20:1",  prev: 2785.58, post: 139.28, src:"peer-2f1a" },
];

// ----- SEARCH RESULTS (peer offers) -----
const SEARCH_RESULTS = [
  { peerId:"peer-7a3f-ec19", rep: 0.94, range:"2018-01-02 → 2026-05-22", updated:"2 min ago",  speed:"42.1 MB/s", rows: 2_064_312, latency: 14, geo:"FRA"  },
  { peerId:"peer-c812-a401", rep: 0.88, range:"2015-06-04 → 2026-05-22", updated:"4 min ago",  speed:"31.7 MB/s", rows: 2_911_488, latency: 22, geo:"NYC"  },
  { peerId:"peer-2f1a-9b3e", rep: 0.82, range:"2020-01-02 → 2026-05-21", updated:"11 min ago", speed:"18.4 MB/s", rows: 1_592_006, latency: 38, geo:"TYO"  },
  { peerId:"peer-9d4e-7c02", rep: 0.71, range:"2019-04-11 → 2026-05-20", updated:"38 min ago", speed:"24.6 MB/s", rows: 1_804_900, latency: 41, geo:"AMS"  },
  { peerId:"peer-44b8-2d10", rep: 0.66, range:"2021-09-13 → 2026-05-22", updated:"1 hr ago",   speed:"12.1 MB/s", rows:   908_220, latency: 64, geo:"SGP"  },
  { peerId:"peer-1c0a-58ef", rep: 0.41, range:"2022-02-01 → 2026-04-28", updated:"3 hr ago",   speed:" 6.2 MB/s", rows:   612_341, latency: 142,geo:"SAO"  },
];

// ----- TRANSFERS -----
const TRANSFERS = [
  { id:"tx-9182", sym:"AAPL", type:"EOD",          src:"peer-7a3f", dest:"local", progress: 78, status:"transferring", speed: 4624128, size: "184 MB", eta:"00:00:24", dir:"in" },
  { id:"tx-9183", sym:"NVDA", type:"EOD",          src:"peer-c812", dest:"local", progress: 42, status:"transferring", speed: 2812416, size: "212 MB", eta:"00:01:18", dir:"in" },
  { id:"tx-9184", sym:"TSLA", type:"INSIDER_TRADE",src:"peer-2f1a", dest:"local", progress: 12, status:"transferring", speed:  984472, size: " 38 MB", eta:"00:00:42", dir:"in" },
  { id:"tx-9185", sym:"MSFT", type:"DIVIDEND",     src:"local",     dest:"peer-9d4e", progress: 88, status:"transferring", speed: 1840128, size: " 28 MB", eta:"00:00:09", dir:"out" },
  { id:"tx-9180", sym:"AMZN", type:"EOD",          src:"peer-44b8", dest:"local", progress:100, status:"completed",    speed:0,           size: "162 MB", eta:"—",         dir:"in" },
  { id:"tx-9181", sym:"META", type:"EOD",          src:"peer-1c0a", dest:"local", progress: 64, status:"failed",       speed:0,           size: " 92 MB", eta:"—",         dir:"in" },
  { id:"tx-9179", sym:"GOOGL",type:"DIVIDEND",     src:"local",     dest:"peer-7a3f", progress:100, status:"completed",speed:0,           size: " 12 MB", eta:"—",         dir:"out" },
  { id:"tx-9178", sym:"BRK.B",type:"EOD",          src:"local",     dest:"peer-2f1a", progress:100, status:"completed",speed:0,           size: " 64 MB", eta:"—",         dir:"out" },
  { id:"tx-9177", sym:"AMD",  type:"EOD",          src:"peer-c812", dest:"local",     progress:  0, status:"pending",  speed:0,           size:"244 MB", eta:"—",          dir:"in" },
];

// ----- SCRIPTS -----
const SCRIPTS = [
  { id:"scr-001", name:"yfinance_eod_v3.py",       dataType:"EOD",          source:"yfinance",     schedule:"0 21 * * 1-5", lastRun: days(0.04), nextRun: days(-0.03), status:"running",   votes:124, isInstalled:true,  author:"peer-7a3f" },
  { id:"scr-002", name:"sec_form4_scraper.py",     dataType:"INSIDER_TRADE",source:"sec.gov",      schedule:"30 6 * * *",   lastRun: days(0.2),  nextRun: days(-0.8), status:"scheduled", votes: 88, isInstalled:true,  author:"peer-c812" },
  { id:"scr-003", name:"dividend_calendar.py",     dataType:"DIVIDEND",     source:"nasdaq.com",   schedule:"0 4 * * 1",    lastRun: days(2.1),  nextRun: days(-5),   status:"idle",      votes: 62, isInstalled:true,  author:"peer-2f1a" },
  { id:"scr-004", name:"splits_history.py",        dataType:"SPLIT",        source:"sharadar",     schedule:"0 5 * * 6",    lastRun: days(1.5),  nextRun: days(-5.5), status:"failed",    votes:-12, isInstalled:true,  author:"peer-9d4e" },
  { id:"scr-005", name:"polygon_eod_premium.py",   dataType:"EOD",          source:"polygon.io",   schedule:"15 21 * * 1-5",lastRun: null,       nextRun: null,       status:"available", votes:241, isInstalled:false, author:"peer-7a3f" },
  { id:"scr-006", name:"alphavantage_dividends.py",dataType:"DIVIDEND",     source:"alphavantage", schedule:"0 5 * * *",    lastRun: null,       nextRun: null,       status:"available", votes:198, isInstalled:false, author:"peer-c812" },
  { id:"scr-007", name:"finra_insider_xbrl.py",    dataType:"INSIDER_TRADE",source:"finra.org",    schedule:"0 7 * * 1-5",  lastRun: null,       nextRun: null,       status:"available", votes: 76, isInstalled:false, author:"peer-44b8" },
];

// ----- PEERS -----
const PEERS = [
  { id:"peer-7a3f-ec19-b884", addr:"45.79.142.18:4001",   rep:0.94, conn:true,  roles:["provider","validator","relay"], lastSeen: mins(0.2),  geo:"FRA", rttMs: 14,  uptime:"99.4%", shared:"2,064k rows" },
  { id:"peer-c812-a401-3d10", addr:"104.131.92.41:4001",  rep:0.88, conn:true,  roles:["provider","relay"],             lastSeen: mins(0.4),  geo:"NYC", rttMs: 22,  uptime:"98.1%", shared:"2,911k rows" },
  { id:"peer-2f1a-9b3e-6c01", addr:"139.99.74.222:4001",  rep:0.82, conn:true,  roles:["provider"],                     lastSeen: mins(1.1),  geo:"TYO", rttMs: 38,  uptime:"96.7%", shared:"1,592k rows" },
  { id:"peer-9d4e-7c02-aa18", addr:"167.71.184.91:4001",  rep:0.71, conn:true,  roles:["provider","validator"],         lastSeen: mins(2.4),  geo:"AMS", rttMs: 41,  uptime:"94.2%", shared:"1,804k rows" },
  { id:"peer-44b8-2d10-9f4c", addr:"159.65.221.10:4001",  rep:0.66, conn:true,  roles:["provider"],                     lastSeen: mins(0.9),  geo:"SGP", rttMs: 64,  uptime:"91.8%", shared:"  908k rows" },
  { id:"peer-1c0a-58ef-7b22", addr:"206.81.14.4:4001",    rep:0.41, conn:true,  roles:["consumer"],                     lastSeen: mins(5.6),  geo:"SAO", rttMs: 142, uptime:"82.4%", shared:"  612k rows" },
  { id:"peer-5e02-c918-0a44", addr:"194.195.114.88:4001", rep:0.32, conn:false, roles:["consumer"],                     lastSeen: mins(28),   geo:"LON", rttMs: 0,   uptime:"71.0%", shared:"      0 rows" },
  { id:"peer-9090-7711-fa01", addr:"96.126.103.12:4001",  rep:0.21, conn:false, roles:["consumer"],                     lastSeen: mins(112),  geo:"SYD", rttMs: 0,   uptime:"64.8%", shared:"      0 rows" },
];

// ----- LOG STRIP (recent events) -----
const LOG_LINES = [
  { ts:"21:04:11", lvl:"info", msg:"DHT bootstrap complete · 47 peers reachable" },
  { ts:"21:04:09", lvl:"ok",   msg:"tx-9180 AMZN.EOD completed · 162 MB · 4.2s" },
  { ts:"21:04:02", lvl:"info", msg:"verified AAPL.EOD against peer-c812 · ✓ match" },
  { ts:"21:03:57", lvl:"warn", msg:"peer-1c0a-58ef latency 142ms · throttling" },
  { ts:"21:03:48", lvl:"err",  msg:"tx-9181 META.EOD failed · checksum mismatch" },
  { ts:"21:03:31", lvl:"info", msg:"scr-001 yfinance_eod_v3 → 38 symbols updated" },
  { ts:"21:03:14", lvl:"ok",   msg:"signed offer · NVDA.EOD · range 2018-01-02 → 2026-05-22" },
];

// 3-day mini sparkline series for the metric tiles
const SPARK = {
  peers:     [38,40,41,40,42,44,43,45,46,47],
  transfers: [12,11,10,11,9,10,11,9,10, 9],
  datasets:  [1140,1158,1180,1198,1214,1230,1248,1262,1276,1284],
  symbols:   [3088,3094,3104,3110,3118,3124,3128,3134,3140,3142],
  uptime:    [99.91,99.92,99.92,99.93,99.93,99.94,99.94,99.94,99.94,99.94],
  hitrate:   [82.1,83.0,82.8,83.6,84.4,84.9,85.3,85.6,86.0,86.2],
};

window.MOCK = {
  TICKER, KPIS, EOD, DIVIDENDS, INSIDER, SPLITS,
  SEARCH_RESULTS, TRANSFERS, SCRIPTS, PEERS,
  LOG_LINES, SPARK,
};
