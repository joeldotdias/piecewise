import { useState, useEffect, useRef, useCallback } from "react";

const PIECE_COUNT = 207;
const SEEDER_PORT = 6881;
const LEECHER_PORTS = [6882, 6883, 6884];

const SOURCE_COLORS = {
    pending: { bg: "bg-zinc-800", border: "border-zinc-700" },
    seeder: { bg: "bg-cyan-500", border: "border-cyan-400" },
    peer: { bg: "bg-amber-400", border: "border-amber-300" },
    done: { bg: "bg-emerald-500", border: "border-emerald-400" },
};

function useTorrentSim(running) {
    const [seederPieces] = useState(() => new Array(PIECE_COUNT).fill("done"));
    const [leechers, setLeechers] = useState(() =>
        LEECHER_PORTS.map((port, i) => ({
            port,
            pieces: new Array(PIECE_COUNT).fill("pending"),
            peers: [],
            log: [`[boot] leecher :${port} started`],
            speed: 0,
            done: 0,
        }))
    );
    const [trackerPeers, setTrackerPeers] = useState([
        { addr: `192.168.0.107:${SEEDER_PORT}`, role: "seeder", since: "just now" },
    ]);
    const [tick, setTick] = useState(0);
    const tickRef = useRef(0);

    const pushLog = useCallback((li, msg) => {
        setLeechers(prev =>
            prev.map((l, i) =>
                i === li
                    ? { ...l, log: [`[${new Date().toLocaleTimeString()}] ${msg}`, ...l.log].slice(0, 40) }
                    : l
            )
        );
    }, []);

    useEffect(() => {
        if (!running) return;

        // leecher 0 joins tracker after 600ms
        const t1 = setTimeout(() => {
            setTrackerPeers(p => [...p, { addr: `192.168.0.107:6882`, role: "leecher", since: "just now" }]);
            pushLog(0, "announced to tracker, found 1 peer");
        }, 600);

        // leecher 1 joins tracker after 1200ms
        const t2 = setTimeout(() => {
            setTrackerPeers(p => [...p, { addr: `192.168.0.107:6883`, role: "leecher", since: "just now" }]);
            pushLog(1, "announced to tracker, found 2 peers");
        }, 1200);

        // leecher 2 joins tracker after 2000ms
        const t3 = setTimeout(() => {
            setTrackerPeers(p => [...p, { addr: `192.168.0.107:6884`, role: "leecher", since: "just now" }]);
            pushLog(2, "announced to tracker, found 3 peers");
        }, 2000);

        return () => { clearTimeout(t1); clearTimeout(t2); clearTimeout(t3); };
    }, [running, pushLog]);

    useEffect(() => {
        if (!running) return;
        const interval = setInterval(() => {
            tickRef.current += 1;
            const t = tickRef.current;
            setTick(t);

            setLeechers(prev => prev.map((l, li) => {
                // stagger start: leecher i starts at tick (li * 3) + 2
                if (t < (li * 3) + 2) return l;

                const newPieces = [...l.pieces];
                const piecesPerTick = li === 0 ? 4 : li === 1 ? 3 : 2;
                let downloaded = 0;
                let newLogs = [];

                for (let p = 0; p < piecesPerTick; p++) {
                    // find a random pending piece
                    const pending = [];
                    for (let i = 0; i < PIECE_COUNT; i++) {
                        if (newPieces[i] === "pending") pending.push(i);
                    }
                    if (pending.length === 0) break;

                    const idx = pending[Math.floor(Math.random() * pending.length)];

                    // leecher 0 gets from seeder only
                    // leecher 1 gets mix: 60% seeder, 40% from leecher 0 if it has pieces
                    // leecher 2 gets mix: 40% seeder, 60% from peers
                    let source = "seeder";
                    if (li === 1 && Math.random() < 0.4) {
                        const l0 = prev[0];
                        if (l0.done > 20) source = "peer";
                    }
                    if (li === 2) {
                        const roll = Math.random();
                        if (roll < 0.5 && prev[0].done > 30) source = "peer";
                        else if (roll < 0.7 && prev[1].done > 15) source = "peer";
                    }

                    newPieces[idx] = source;
                    downloaded++;
                    newLogs.push(`piece ${idx} ← ${source === "peer" ? "peer" : `seeder:${SEEDER_PORT}`}`);
                }

                const done = newPieces.filter(p => p !== "pending").length;
                const speed = Math.round((downloaded * 16384) / 1024 * (0.8 + Math.random() * 0.4));

                return {
                    ...l,
                    pieces: newPieces,
                    done,
                    speed: done === PIECE_COUNT ? 0 : speed,
                    log: [...newLogs.map(m => `[${new Date().toLocaleTimeString()}] ${m}`), ...l.log].slice(0, 40),
                };
            }));
        }, 180);

        return () => clearInterval(interval);
    }, [running]);

    return { seederPieces, leechers, trackerPeers };
}

function PieceGrid({ pieces, total = PIECE_COUNT }) {
    return (
        <div className="flex flex-wrap gap-[2px]">
            {Array.from({ length: total }).map((_, i) => {
                const state = pieces[i] ?? "pending";
                const c = SOURCE_COLORS[state] ?? SOURCE_COLORS.pending;
                return (
                    <div
                        key={i}
                        title={`piece ${i} — ${state}`}
                        className={`w-2 h-2 rounded-sm border transition-all duration-300 ${c.bg} ${c.border}`}
                    />
                );
            })}
        </div>
    );
}

function LogPanel({ log }) {
    const ref = useRef(null);
    return (
        <div ref={ref} className="h-24 overflow-y-auto font-mono text-[10px] text-zinc-400 space-y-[2px] pr-1">
            {log.map((line, i) => (
                <div key={i} className={i === 0 ? "text-zinc-200" : ""}>{line}</div>
            ))}
        </div>
    );
}

function NodeCard({ title, port, role, pieces, done, speed, log, peers, isSeeder }) {
    const pct = isSeeder ? 100 : Math.round((done / PIECE_COUNT) * 100);
    const isDone = pct === 100;

    return (
        <div className={`rounded-xl border p-4 flex flex-col gap-3 transition-all duration-500
      ${isSeeder
                ? "border-cyan-800 bg-zinc-900"
                : isDone
                    ? "border-emerald-700 bg-zinc-900"
                    : "border-zinc-700 bg-zinc-900/80"
            }`}>

            {/* header */}
            <div className="flex items-center justify-between">
                <div>
                    <span className="font-mono text-sm font-bold text-zinc-100">{title}</span>
                    <span className="ml-2 font-mono text-xs text-zinc-500">:{port}</span>
                </div>
                <span className={`text-[10px] font-mono px-2 py-0.5 rounded-full border
          ${isSeeder
                        ? "text-cyan-400 border-cyan-700 bg-cyan-950"
                        : isDone
                            ? "text-emerald-400 border-emerald-700 bg-emerald-950"
                            : done > 0
                                ? "text-amber-400 border-amber-700 bg-amber-950"
                                : "text-zinc-500 border-zinc-700 bg-zinc-800"
                    }`}>
                    {isSeeder ? "seeding" : isDone ? "done" : done > 0 ? "leeching" : "waiting"}
                </span>
            </div>

            {/* progress bar */}
            <div className="space-y-1">
                <div className="flex justify-between font-mono text-[10px] text-zinc-500">
                    <span>{isSeeder ? "207 / 207 pieces" : `${done} / ${PIECE_COUNT} pieces`}</span>
                    <span>{isSeeder ? "—" : `${speed} KB/s`}</span>
                </div>
                <div className="h-1.5 w-full bg-zinc-800 rounded-full overflow-hidden">
                    <div
                        className={`h-full rounded-full transition-all duration-300
              ${isSeeder ? "bg-cyan-500" : isDone ? "bg-emerald-500" : "bg-amber-400"}`}
                        style={{ width: `${pct}%` }}
                    />
                </div>
            </div>

            {/* piece grid */}
            <PieceGrid pieces={pieces} />

            {/* log */}
            <div className="border-t border-zinc-800 pt-2">
                <LogPanel log={log} />
            </div>
        </div>
    );
}

function TrackerCard({ peers }) {
    return (
        <div className="rounded-xl border border-zinc-700 bg-zinc-900/80 p-4 flex flex-col gap-3">
            <div className="flex items-center justify-between">
                <span className="font-mono text-sm font-bold text-zinc-100">tracker</span>
                <span className="text-[10px] font-mono px-2 py-0.5 rounded-full border text-green-400 border-green-800 bg-green-950">
                    online :8080
                </span>
            </div>
            <div className="space-y-1.5">
                {peers.map((p, i) => (
                    <div key={i} className="flex items-center justify-between font-mono text-[11px]">
                        <span className="text-zinc-300">{p.addr}</span>
                        <span className={`px-1.5 py-0.5 rounded text-[10px]
              ${p.role === "seeder"
                                ? "text-cyan-400 bg-cyan-950"
                                : "text-amber-400 bg-amber-950"
                            }`}>
                            {p.role}
                        </span>
                    </div>
                ))}
            </div>
            <div className="border-t border-zinc-800 pt-2 font-mono text-[10px] text-zinc-500 space-y-1">
                <div className="flex justify-between"><span>swarm size</span><span className="text-zinc-300">{peers.length}</span></div>
                <div className="flex justify-between"><span>total pieces</span><span className="text-zinc-300">{PIECE_COUNT}</span></div>
                <div className="flex justify-between"><span>announce interval</span><span className="text-zinc-300">1800s</span></div>
            </div>
        </div>
    );
}

export default function TorrentDashboard() {
    const [running, setRunning] = useState(false);
    const { seederPieces, leechers, trackerPeers } = useTorrentSim(running);

    const allDone = leechers.every(l => l.done === PIECE_COUNT);

    return (
        <div className="min-h-screen bg-zinc-950 text-zinc-100 p-6 font-mono">
            {/* header */}
            <div className="mb-6 flex items-end justify-between">
                <div>
                    <h1 className="text-xl font-bold tracking-tight text-zinc-100">
                        bittorrent <span className="text-cyan-400">//</span> lan demo
                    </h1>
                    <p className="text-xs text-zinc-500 mt-0.5">
                        ADS_Module2.pdf · 207 pieces · 256 KB each · 192.168.0.107
                    </p>
                </div>
                <div className="flex items-center gap-3">
                    {/* legend */}
                    <div className="flex items-center gap-3 text-[10px] text-zinc-400 mr-2">
                        {[
                            { color: "bg-cyan-500", label: "from seeder" },
                            { color: "bg-amber-400", label: "from peer" },
                            { color: "bg-emerald-500", label: "verified" },
                            { color: "bg-zinc-800", label: "pending" },
                        ].map(({ color, label }) => (
                            <div key={label} className="flex items-center gap-1">
                                <div className={`w-2 h-2 rounded-sm ${color}`} />
                                <span>{label}</span>
                            </div>
                        ))}
                    </div>
                    <button
                        onClick={() => setRunning(r => !r)}
                        className={`px-4 py-1.5 rounded-lg text-xs font-bold border transition-all
              ${running
                                ? "border-red-700 text-red-400 bg-red-950 hover:bg-red-900"
                                : "border-cyan-700 text-cyan-400 bg-cyan-950 hover:bg-cyan-900"
                            }`}
                    >
                        {running ? (allDone ? "✓ done" : "■ stop") : "▶ simulate"}
                    </button>
                </div>
            </div>

            {/* grid: tracker + seeder + leechers */}
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-5 gap-4">
                {/* tracker — 1 col */}
                <div className="xl:col-span-1">
                    <TrackerCard peers={trackerPeers} />
                </div>

                {/* seeder — 1 col */}
                <div className="xl:col-span-1">
                    <NodeCard
                        title="seeder"
                        port={SEEDER_PORT}
                        role="seeder"
                        pieces={seederPieces}
                        done={PIECE_COUNT}
                        speed={0}
                        log={["[boot] seed initialized", "[boot] all 207 pieces verified", "[boot] listening for peers"]}
                        isSeeder={true}
                    />
                </div>

                {/* leechers — 3 cols */}
                {leechers.map((l, i) => (
                    <div key={l.port} className="xl:col-span-1">
                        <NodeCard
                            title={`leecher ${i + 1}`}
                            port={l.port}
                            role="leecher"
                            pieces={l.pieces}
                            done={l.done}
                            speed={l.speed}
                            log={l.log}
                            isSeeder={false}
                        />
                    </div>
                ))}
            </div>
        </div>
    );
}
