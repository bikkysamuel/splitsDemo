#!/usr/bin/env python3
"""Writes docs/conversations/conversation-in-order.md: every prompt the user
sent and every question Claude asked, oldest first, from the Claude Code
session transcripts in ~/.claude/projects/<this repo>/. Run from anywhere."""
import json, glob, os, re
ROOT=os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
PROJ=os.path.expanduser("~/.claude/projects/"+re.sub(r"[^A-Za-z0-9]","-",ROOT))
OUT=os.path.join(ROOT,"docs","conversations","conversation-in-order.md")
events={}  # (ts, kind) -> payload ; dedupe across forked sessions
ask_ids=set()  # AskUserQuestion tool calls, so only their results count as answers
def clean_user(t):
    if any(k in t for k in ("<local-command-stdout>","<local-command-caveat>","<task-notification>")) or t.startswith(("<system-reminder>","[SYSTEM NOTIFICATION","Another Claude session","Base directory for this skill")): return None
    m=re.search(r"<command-name>(.*?)</command-name>",t)
    if m:
        a=re.search(r"<command-args>(.*?)</command-args>",t,re.S)
        t=m.group(1)+((" "+a.group(1).strip()) if a and a.group(1).strip() else "")
    return t.strip()
for f in glob.glob(PROJ+"/*.jsonl"):
    last_text=None
    for line in open(f):
        try: r=json.loads(line)
        except: continue
        ts=r.get("timestamp","")[:19]
        if r.get("type")=="assistant":
            for it in r["message"].get("content",[]):
                if not isinstance(it,dict): continue
                if it.get("type")=="text" and it["text"].strip():
                    last_text=(ts,it["text"])
                elif it.get("type")=="tool_use" and it.get("name")=="AskUserQuestion":
                    events[(ts,"ask")]=it["input"].get("questions",[])
                    ask_ids.add(it.get("id"))
                    last_text=None
        elif r.get("type")=="user" and not r.get("isMeta"):
            c=r["message"].get("content")
            items=[{"type":"text","text":c}] if isinstance(c,str) else (c or [])
            for it in items:
                if it.get("type")=="text":
                    t=clean_user(it["text"])
                    if t:
                        if last_text and t!="/clear":
                            qs=[l.strip() for l in re.split(r"\n+",last_text[1]) if l.strip().endswith("?") and not l.strip().startswith(("`","|","```"))]
                            if qs: events[(last_text[0],"text-q")]=qs
                        events[(ts,"you")]=t; last_text=None
                elif it.get("type")=="tool_result" and it.get("tool_use_id") in ask_ids:
                    cc=it.get("content"); s=cc if isinstance(cc,str) else " ".join(x.get("text","") for x in (cc or []) if isinstance(x,dict))
                    if "questions have been answered" in s:
                        ans=dict(re.findall(r'"((?:[^"\\]|\\.)*)"="((?:[^"\\]|\\.)*)"',s))
                        events[(ts,"answer")]=ans

# ---- render ----
def quote(t): return "\n".join("> "+l if l.strip() else ">" for l in t.split("\n"))
out=["# The conversation, in order","",
 "Every prompt you sent and every question Claude asked you, oldest first, across all sessions. Generated from the Claude Code session transcripts by `scripts/conversation-in-order.py`; rerun it to bring this file up to date. The task-by-task logs with decisions are in the dated folders next to this file.",""]
day=None
for (ts,kind) in sorted(events):
    v=events[(ts,kind)]
    if ts[:10]!=day:
        day=ts[:10]; out+=[f"## {day} (UTC)",""]
    hhmm=ts[11:16]
    if kind=="you":
        if v=="/clear":
            out+=[f"**{hhmm} · You:** `/clear` (new session)",""]; continue
        out+=[f"**{hhmm} · You:**",quote(v),""]
    elif kind=="text-q":
        out+=[f"**{hhmm} · Claude asked:**"]+[f"- {q.lstrip('-•* ').strip()}" for q in v]+[""]
    elif kind=="ask":
        out+=[f"**{hhmm} · Claude asked (multiple choice):**"]
        for i,q in enumerate(v,1):
            opts=" · ".join(o["label"] for o in q.get("options",[]))
            out+=[f"{i}. {q['question']}",f"   - Options: {opts}"]
        out+=[""]
    elif kind=="answer":
        out+=[f"**{hhmm} · You answered:**"]+[f"- {a}" for a in v.values()]+[""]
open(OUT,"w").write("\n".join(out).rstrip()+"\n")
