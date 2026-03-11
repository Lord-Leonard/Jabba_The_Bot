// package.json
// {
//   "type": "module"
// }
//
// Run:
//   node scripts/next.js
//
// Optional:
//   set INNERTUBE_API_KEY=...   (Windows cmd)
//   $env:INNERTUBE_API_KEY="..." (PowerShell)
//
// This script runs a matrix of /next requests to isolate which fields
// actually influence queue results in logged-out WEB_REMIX context.

import { createHash } from "node:crypto";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const ORIGIN = "https://music.youtube.com";
const INNERTUBE_API_KEY =
    process.env.INNERTUBE_API_KEY ?? "AIzaSyD_Owy59_EfAHrX36X301Q-897dV9_ayRo";
const OUT_DIR = path.join(__dirname, "next-matrix-runs");

const CAPTURED_SIGNALS = [
    { queueImpress: {}, videoId: "jGnX4mXLLJw", queueIndex: 0 },
    { playbackSkip: {}, videoId: "jGnX4mXLLJw", queueIndex: 0 },
    { queueImpress: {}, videoId: "I8C1IhXdPt4", queueIndex: 1 },
    { queueImpress: {}, videoId: "uRbhg1JvqZg", queueIndex: 2 },
];

const DOUBLE_SKIP_SIGNALS = [
    { queueImpress: {}, videoId: "jGnX4mXLLJw", queueIndex: 0 },
    { playbackSkip: {}, videoId: "jGnX4mXLLJw", queueIndex: 0 },
    { queueImpress: {}, videoId: "I8C1IhXdPt4", queueIndex: 1 },
    { playbackSkip: {}, videoId: "I8C1IhXdPt4", queueIndex: 1 },
    { queueImpress: {}, videoId: "uRbhg1JvqZg", queueIndex: 2 },
];

const BASE_PAYLOAD = {
    enablePersistentPlaylistPanel: true,
    tunerSettingValue: "AUTOMIX_SETTING_NORMAL",
    playlistId: "RDATiXvjGnX4mXLLJw",
    params:
        "ggVAQ2hKU1JFRlVhVmgyYWtkdVdEUnRXRXhNU25jWUN5SVRDZzFPWlhVZ1pXNTBaR1ZqYTJWdUdnSmtaUSUzRCUzRA%3D%3D",
    loggingContext: {
        vssLoggingContext: {
            serializedContextData: "GhJSREFUaVh2akduWDRtWExMSnc%3D",
        },
    },
    index: 1,
    isAudioOnly: true,
    responsiveSignals: {
        videoInteraction: CAPTURED_SIGNALS,
    },
    queueContextParams:
        "CAIaEVJEQU1WTWpHblg0bVhMTEp3IJ7S1oOClZMDMgt1UmJoZzFKdnFaZzoQMDE3MjA4RkFBODUyMzNGOUIRUkRBTVZNakduWDRtWExMSndQAVoECAAQAWIAcAF4AQ==",
    context: {
        client: {
            hl: "de",
            gl: "DE",
            remoteHost: "2003:e9:2720:57ab:34:4e35:9af:77a0",
            deviceMake: "",
            deviceModel: "",
            visitorData: "CgswYkdXa2Fua3FmTSjGx7_NBjIKCgJERRIEEgAgQA%3D%3D",
            userAgent:
                "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36,gzip(gfe)",
            clientName: "WEB_REMIX",
            clientVersion: "1.20260304.03.00",
            osName: "Windows",
            osVersion: "10.0",
            originalUrl: "https://music.youtube.com/?cbrd=1",
            platform: "DESKTOP",
            clientFormFactor: "UNKNOWN_FORM_FACTOR",
            configInfo: {
                appInstallData:
                    "CMbHv80GEPCdzxwQy_nQHBDQwoATEPyyzhwQgc3OHBDI988cELyk0BwQzN-uBRCfz4ATENvU0BwQpuvQHBCZ-tAcEOvm0BwQprbQHBDa984cEIv3zxwQvZmwBRC9irAFEMGq0BwQmN3QHBDGxs8cEJS20BwQndCAExC45M4cEImwzhwQlP6wBRDevM4cEJmNsQUQvbauBRCUg9AcEJn50BwQ_PbQHBDU_9AcEK7WzxwQh6zOHBDBj9AcEIzpzxwQ9quwBRDE0oATEJX3zxwQu9nOHBDxtNAcENTu0BwQntCwBRD389AcKkRDQU1TTHhVcy1acS1ETGlVRXBRQ25BNzVGWjBGNUxMd0N6S19YLW5WQlFQTl93WEx0d2I3SnJia0JzbVZCTV84QVIwSDAA",
            },
            utcOffsetMinutes: 60,
            timeZone: "Europe/Berlin",
            browserName: "Chrome",
            browserVersion: "145.0.0.0",
            screenWidthPoints: 958,
            screenHeightPoints: 945,
            screenPixelDensity: 2,
        },
        user: {
            lockedSafetyMode: false,
        },
        request: {
            useSsl: true,
            internalExperimentFlags: [],
            consistencyTokenJars: [],
        },
    },
};

function deepClone(value) {
    return JSON.parse(JSON.stringify(value));
}

function setSignals(payload, interactions) {
    payload.responsiveSignals = { videoInteraction: interactions };
}

function omitSignals(payload) {
    delete payload.responsiveSignals;
}

function extractCurrent(data) {
    const w = data?.currentVideoEndpoint?.watchEndpoint;
    if (!w) return null;

    return {
        videoId: w.videoId ?? null,
        playlistId: w.playlistId ?? null,
        index: w.index ?? null,
        playlistSetVideoId: w.playlistSetVideoId ?? null,
        playerParams: w.playerParams ?? null,
    };
}

function extractItem(node) {
    const r = node?.playlistPanelVideoRenderer;
    if (!r) return null;

    return {
        title: r.title?.runs?.map((x) => x.text).join("") ?? null,
        videoId: r.videoId ?? r.navigationEndpoint?.watchEndpoint?.videoId ?? null,
        index: r.navigationEndpoint?.watchEndpoint?.index ?? null,
        selected: Boolean(r.selected),
        playlistSetVideoId:
            r.playlistSetVideoId ??
            r.navigationEndpoint?.watchEndpoint?.playlistSetVideoId ??
            null,
    };
}

function extractPrimaryQueue(data) {
    const tabs =
        data?.contents?.singleColumnMusicWatchNextResultsRenderer?.tabbedRenderer
            ?.watchNextTabbedResultsRenderer?.tabs ?? [];

    for (const tab of tabs) {
        const contents =
            tab?.tabRenderer?.content?.musicQueueRenderer?.content?.playlistPanelRenderer
                ?.contents;
        if (Array.isArray(contents) && contents.length > 0) {
            return contents.map(extractItem).filter(Boolean);
        }
    }

    const out = [];
    function walk(node) {
        if (!node || typeof node !== "object") return;
        if (Array.isArray(node)) {
            for (const item of node) walk(item);
            return;
        }

        const item = extractItem(node);
        if (item) out.push(item);

        for (const value of Object.values(node)) walk(value);
    }

    walk(data);
    return out;
}

function queueKey(x) {
    return [
        x.videoId ?? "-",
        x.playlistSetVideoId ?? "-",
        String(x.index ?? "-"),
        x.selected ? "1" : "0",
    ].join("|");
}

function currentKey(x) {
    if (!x) return "-";
    return [
        x.videoId ?? "-",
        x.playlistId ?? "-",
        String(x.index ?? "-"),
        x.playlistSetVideoId ?? "-",
    ].join("|");
}

function hashRows(rows) {
    return createHash("sha256").update(rows.join("\n")).digest("hex").slice(0, 16);
}

function firstDiffPositions(baseQueue, nextQueue, limit = 50, maxFindings = 8) {
    const findings = [];
    const max = Math.min(Math.max(baseQueue.length, nextQueue.length), limit);

    for (let i = 0; i < max; i++) {
        const left = baseQueue[i] ?? null;
        const right = nextQueue[i] ?? null;
        const leftKey = queueKey(left ?? {});
        const rightKey = queueKey(right ?? {});
        if (leftKey !== rightKey) {
            findings.push({
                pos: i,
                base: left?.videoId ?? "-",
                next: right?.videoId ?? "-",
                baseKey: leftKey,
                nextKey: rightKey,
            });
            if (findings.length >= maxFindings) break;
        }
    }

    return findings;
}

function topVideoIds(queue, n = 8) {
    return queue.slice(0, n).map((x) => x.videoId ?? "-");
}

async function saveArtifacts(scenarioId, requestPayload, responseData) {
    await mkdir(OUT_DIR, { recursive: true });
    const base = path.join(OUT_DIR, scenarioId);
    await writeFile(`${base}.request.json`, JSON.stringify(requestPayload, null, 2));
    await writeFile(`${base}.response.json`, JSON.stringify(responseData, null, 2));
}

function scenarioId(label, index) {
    const slug = label
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, "-")
        .replace(/^-+|-+$/g, "");
    return `${String(index + 1).padStart(2, "0")}-${slug}`;
}

async function callNext({ label, payload, id }) {
    const url = `${ORIGIN}/youtubei/v1/next?prettyPrint=false&key=${encodeURIComponent(INNERTUBE_API_KEY)}`;
    const headers = {
        "content-type": "application/json",
        origin: ORIGIN,
        referer: `${ORIGIN}/watch?v=uRbhg1JvqZg&list=RDATiXvjGnX4mXLLJw`,
        "user-agent":
            "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36",
        "x-goog-authuser": "0",
        "x-origin": ORIGIN,
        "x-youtube-client-name": "67",
        "x-youtube-client-version": payload?.context?.client?.clientVersion ?? "1.20260304.03.00",
    };

    let res;
    try {
        res = await fetch(url, {
            method: "POST",
            headers,
            body: JSON.stringify(payload),
        });
    } catch (err) {
        return {
            label,
            id,
            ok: false,
            httpStatus: null,
            error: `network_error: ${err?.message ?? String(err)}`,
        };
    }

    const text = await res.text();
    let data = null;
    try {
        data = JSON.parse(text);
    } catch {
        return {
            label,
            id,
            ok: false,
            httpStatus: res.status,
            error: "non_json_response",
            bodyPreview: text.slice(0, 400),
        };
    }

    if (!res.ok) {
        return {
            label,
            id,
            ok: false,
            httpStatus: res.status,
            error: "http_error",
            data,
        };
    }

    const queue = extractPrimaryQueue(data);
    const current = extractCurrent(data);
    const queueRows = queue.map(queueKey);
    const top10Hash = hashRows(queueRows.slice(0, 10));
    const fullHash = hashRows(queueRows);

    await saveArtifacts(id, payload, data);

    return {
        label,
        id,
        ok: true,
        httpStatus: res.status,
        raw: data,
        queue,
        current,
        queueLen: queue.length,
        top10Hash,
        fullHash,
        currentSignature: currentKey(current),
        responseVisitorData: data?.responseContext?.visitorData ?? null,
        responseQueueContextParams: data?.queueContextParams ?? null,
    };
}

const SCENARIOS = [
    {
        label: "control: captured request",
        mutate(payload) {
            setSignals(payload, CAPTURED_SIGNALS);
        },
    },
    {
        label: "omit responsiveSignals",
        mutate(payload) {
            omitSignals(payload);
        },
    },
    {
        label: "responsiveSignals: []",
        mutate(payload) {
            setSignals(payload, []);
        },
    },
    {
        label: "double skip signals",
        mutate(payload) {
            setSignals(payload, DOUBLE_SKIP_SIGNALS);
        },
    },
    {
        label: "remove queueContextParams",
        mutate(payload) {
            setSignals(payload, CAPTURED_SIGNALS);
            delete payload.queueContextParams;
        },
    },
    {
        label: "remove params",
        mutate(payload) {
            setSignals(payload, CAPTURED_SIGNALS);
            delete payload.params;
        },
    },
    {
        label: "remove params + queueContextParams",
        mutate(payload) {
            setSignals(payload, CAPTURED_SIGNALS);
            delete payload.params;
            delete payload.queueContextParams;
        },
    },
    {
        label: "index=0",
        mutate(payload) {
            setSignals(payload, CAPTURED_SIGNALS);
            payload.index = 0;
        },
    },
    {
        label: "index=2",
        mutate(payload) {
            setSignals(payload, CAPTURED_SIGNALS);
            payload.index = 2;
        },
    },
    {
        label: "index=3",
        mutate(payload) {
            setSignals(payload, CAPTURED_SIGNALS);
            payload.index = 3;
        },
    },
    {
        label: "use response visitorData from control",
        mutate(payload, state) {
            setSignals(payload, CAPTURED_SIGNALS);
            const control = state.results.find((x) => x.label === "control: captured request");
            if (control?.responseVisitorData) {
                payload.context.client.visitorData = control.responseVisitorData;
            }
        },
    },
    {
        label: "use response queueContextParams from control",
        mutate(payload, state) {
            setSignals(payload, CAPTURED_SIGNALS);
            const control = state.results.find((x) => x.label === "control: captured request");
            if (control?.responseQueueContextParams) {
                payload.queueContextParams = control.responseQueueContextParams;
            }
        },
    },
    {
        label: "use both response visitorData + queueContextParams",
        mutate(payload, state) {
            setSignals(payload, CAPTURED_SIGNALS);
            const control = state.results.find((x) => x.label === "control: captured request");
            if (control?.responseVisitorData) {
                payload.context.client.visitorData = control.responseVisitorData;
            }
            if (control?.responseQueueContextParams) {
                payload.queueContextParams = control.responseQueueContextParams;
            }
        },
    },
];

function printScenario(result) {
    if (!result.ok) {
        console.log(
            `[${result.id}] ${result.label} -> FAIL (${result.httpStatus ?? "no-status"}) ${result.error}`
        );
        if (result.bodyPreview) {
            console.log(`  body: ${result.bodyPreview.replace(/\s+/g, " ").slice(0, 180)}`);
        }
        return;
    }

    console.log(`[${result.id}] ${result.label}`);
    console.log(
        `  HTTP ${result.httpStatus} | queue=${result.queueLen} | top10=${result.top10Hash} | full=${result.fullHash}`
    );
    console.log(`  current=${result.currentSignature}`);
    console.log(`  top8=${topVideoIds(result.queue).join(", ")}`);
}

function printSummary(results) {
    const okResults = results.filter((x) => x.ok);
    if (okResults.length === 0) return;

    const base = okResults[0];
    const rows = okResults.map((r) => {
        const sameTop10 = r.top10Hash === base.top10Hash;
        const sameFull = r.fullHash === base.fullHash;
        const sameCurrent = r.currentSignature === base.currentSignature;
        const firstDiffs = firstDiffPositions(base.queue, r.queue);

        return {
            scenario: r.id,
            sameTop10: sameTop10 ? "yes" : "no",
            sameFull: sameFull ? "yes" : "no",
            sameCurrent: sameCurrent ? "yes" : "no",
            queueLen: r.queueLen,
            firstDiff: firstDiffs.length
                ? `${firstDiffs[0].pos}:${firstDiffs[0].base}->${firstDiffs[0].next}`
                : "-",
        };
    });

    console.log("\n=== summary vs control ===");
    console.table(rows);

    for (const r of okResults.slice(1)) {
        const diffs = firstDiffPositions(base.queue, r.queue);
        if (diffs.length === 0) continue;
        console.log(`\n[${r.id}] first queue diffs vs control:`);
        for (const d of diffs) {
            console.log(`  pos ${d.pos}: ${d.base} -> ${d.next}`);
            console.log(`    ${d.baseKey}`);
            console.log(`    ${d.nextKey}`);
        }
    }
}

function applyResponseState(payload, result) {
    if (!result?.ok) return;
    if (result.responseVisitorData) {
        payload.context.client.visitorData = result.responseVisitorData;
    }
    if (result.responseQueueContextParams) {
        payload.queueContextParams = result.responseQueueContextParams;
    }
}

function compareResultToControl(control, candidate, label) {
    if (!control?.ok || !candidate?.ok) {
        console.log(`\n=== ${label} ===`);
        console.log("comparison skipped (missing successful result)");
        return;
    }

    const diffs = firstDiffPositions(control.queue, candidate.queue, 60, 10);
    const sameTop10 = control.top10Hash === candidate.top10Hash;
    const sameFull = control.fullHash === candidate.fullHash;
    const sameCurrent = control.currentSignature === candidate.currentSignature;

    console.log(`\n=== ${label} ===`);
    console.log(`sameTop10=${sameTop10} sameFull=${sameFull} sameCurrent=${sameCurrent}`);
    if (diffs.length === 0) {
        console.log("no queue diffs found");
        return;
    }
    for (const d of diffs) {
        console.log(`pos ${d.pos}: ${d.base} -> ${d.next}`);
        console.log(`  ${d.baseKey}`);
        console.log(`  ${d.nextKey}`);
    }
}

async function runDelayedFeedbackProbe(state) {
    const control = state.results.find((x) => x.label === "control: captured request");
    if (!control?.ok) {
        console.log("\n=== delayed feedback probe ===");
        console.log("skipped because control scenario failed");
        return;
    }

    // Step A: push aggressive skip feedback and roll state from control response.
    const stepAPayload = deepClone(BASE_PAYLOAD);
    applyResponseState(stepAPayload, control);
    setSignals(stepAPayload, DOUBLE_SKIP_SIGNALS);
    stepAPayload.index = 2;
    const stepA = await callNext({
        label: "probe A: feedback write attempt",
        id: "14-probe-a-feedback-write-attempt",
        payload: stepAPayload,
    });
    state.results.push(stepA);
    printScenario(stepA);

    // Step B: follow-up request with no signals, carrying state from step A.
    const stepBPayload = deepClone(BASE_PAYLOAD);
    omitSignals(stepBPayload);
    applyResponseState(stepBPayload, stepA.ok ? stepA : control);
    stepBPayload.index = 2;
    const stepB = await callNext({
        label: "probe B: follow-up read after feedback",
        id: "15-probe-b-follow-up-read-after-feedback",
        payload: stepBPayload,
    });
    state.results.push(stepB);
    printScenario(stepB);

    compareResultToControl(control, stepB, "delayed feedback effect vs control");
}

async function main() {
    const state = { results: [] };

    for (let i = 0; i < SCENARIOS.length; i++) {
        const scenario = SCENARIOS[i];
        const id = scenarioId(scenario.label, i);
        const payload = deepClone(BASE_PAYLOAD);
        scenario.mutate(payload, state);

        const result = await callNext({
            label: scenario.label,
            id,
            payload,
        });

        state.results.push(result);
        printScenario(result);
    }

    await runDelayedFeedbackProbe(state);

    printSummary(state.results);
    console.log(`\nartifacts written to: ${OUT_DIR}`);
}

main().catch((err) => {
    console.error(err);
    process.exit(1);
});
