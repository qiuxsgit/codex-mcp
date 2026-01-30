'use client';

import { useState, useEffect, useCallback } from 'react';

const API = '';

type Directory = {
  id: number;
  name: string;
  path: string;
  language: string;
  role: string;
  enabled: boolean;
  git_auto_update_interval_sec: number;
  git_last_updated_at: string | null;
};

const GIT_INTERVALS: { value: number; label: string }[] = [
  { value: 0, label: '关闭' },
  { value: 300, label: '5 分钟' },
  { value: 600, label: '10 分钟' },
  { value: 1800, label: '30 分钟' },
  { value: 3600, label: '1 小时' },
];

const ROLE_OPTIONS = [
  { value: '', label: '请选择' },
  { value: '前端业务', label: '前端业务' },
  { value: '后端业务', label: '后端业务' },
  { value: '前端框架', label: '前端框架' },
  { value: '后端框架', label: '后端框架' },
];

const LANGUAGE_OPTIONS = [
  { value: '', label: '不限' },
  { value: 'java', label: 'Java' },
  { value: 'js', label: 'React / JSX' },
  { value: 'py', label: 'Python' },
  { value: 'go', label: 'Go' },
  { value: 'ts', label: 'TypeScript' },
  { value: 'javascript', label: 'JavaScript' },
  { value: 'csharp', label: 'C#' },
  { value: 'cpp', label: 'C++' },
  { value: 'rust', label: 'Rust' },
  { value: 'vue', label: 'Vue' },
  { value: 'swift', label: 'Swift' },
  { value: 'kotlin', label: 'Kotlin' },
  { value: 'ruby', label: 'Ruby' },
  { value: 'php', label: 'PHP' },
];

function formatGitLastUpdated(iso: string | null): string {
  if (!iso) return '—';
  try {
    return new Date(iso).toLocaleString('zh-CN');
  } catch {
    return iso;
  }
}

async function dirList(): Promise<Directory[]> {
  const r = await fetch(`${API}/api/directories`);
  if (!r.ok) throw new Error('加载失败');
  return r.json();
}

async function dirAdd(name: string, path: string, language: string, role: string) {
  const r = await fetch(`${API}/api/directories`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, path, language, role }),
  });
  if (!r.ok) throw new Error('添加失败');
  return r.json();
}

async function dirDelete(id: number) {
  const r = await fetch(`${API}/api/directories/${id}`, { method: 'DELETE' });
  if (!r.ok) throw new Error('删除失败');
}

async function dirSetEnabled(id: number, enabled: boolean) {
  const r = await fetch(`${API}/api/directories/${id}/enabled`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ enabled }),
  });
  if (!r.ok) throw new Error('更新失败');
}

async function dirSetGitInterval(id: number, intervalSec: number) {
  const r = await fetch(`${API}/api/directories/${id}/git`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ auto_update_interval_sec: intervalSec }),
  });
  if (!r.ok) throw new Error('设置失败');
}

async function dirGitPull(id: number) {
  const r = await fetch(`${API}/api/directories/${id}/git/pull`, { method: 'POST' });
  if (!r.ok) throw new Error('拉取失败');
}

async function ignoreGet(): Promise<string> {
  const r = await fetch(`${API}/api/ignore-file`);
  if (!r.ok) throw new Error('加载失败');
  return r.text();
}

async function ignorePut(content: string) {
  const r = await fetch(`${API}/api/ignore-file`, {
    method: 'PUT',
    body: content,
  });
  if (!r.ok) throw new Error('保存失败');
}

export default function AdminPage() {
  const [dirs, setDirs] = useState<Directory[]>([]);
  const [loading, setLoading] = useState(true);
  const [dirMsg, setDirMsg] = useState<{ type: 'error' | 'success'; text: string } | null>(null);
  const [ignoreContent, setIgnoreContent] = useState('');
  const [ignoreMsg, setIgnoreMsg] = useState<{ type: 'error' | 'success'; text: string } | null>(null);
  const [addName, setAddName] = useState('');
  const [addPath, setAddPath] = useState('');
  const [addLang, setAddLang] = useState('');
  const [addRole, setAddRole] = useState('');
  const [pullingId, setPullingId] = useState<number | null>(null);
  const [ignoreModalOpen, setIgnoreModalOpen] = useState(false);

  const refreshDirs = useCallback(async () => {
    setDirMsg(null);
    try {
      const list = await dirList();
      setDirs(list);
    } catch (e) {
      setDirMsg({ type: 'error', text: String((e as Error).message) });
    }
  }, []);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const list = await dirList();
        if (!cancelled) setDirs(list);
      } catch {
        if (!cancelled) setDirMsg({ type: 'error', text: '加载目录失败' });
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, []);

  const openIgnoreModal = useCallback(() => {
    setIgnoreModalOpen(true);
    setIgnoreMsg(null);
    ignoreGet().then((t) => setIgnoreContent(t || '')).catch(() => {});
  }, []);

  const handleAdd = async () => {
    const name = addName.trim();
    const path = addPath.trim();
    const role = addRole.trim();
    if (!name || !path) {
      setDirMsg({ type: 'error', text: '名称和路径必填' });
      return;
    }
    if (!role) {
      setDirMsg({ type: 'error', text: '请选择角色（前端/后端业务或框架）' });
      return;
    }
    setDirMsg(null);
    try {
      await dirAdd(name, path, addLang.trim(), role);
      setDirMsg({ type: 'success', text: '已添加' });
      setAddName('');
      setAddPath('');
      await refreshDirs();
    } catch (e) {
      setDirMsg({ type: 'error', text: String((e as Error).message) });
    }
  };

  const handleDelete = async (id: number) => {
    if (!confirm('确定删除？')) return;
    setDirMsg(null);
    try {
      await dirDelete(id);
      setDirMsg({ type: 'success', text: '已删除' });
      await refreshDirs();
    } catch (e) {
      setDirMsg({ type: 'error', text: String((e as Error).message) });
    }
  };

  const handleToggle = async (d: Directory) => {
    setDirMsg(null);
    try {
      await dirSetEnabled(d.id, !d.enabled);
      await refreshDirs();
    } catch (e) {
      setDirMsg({ type: 'error', text: String((e as Error).message) });
    }
  };

  const handleGitInterval = async (id: number, intervalSec: number) => {
    setDirMsg(null);
    try {
      await dirSetGitInterval(id, intervalSec);
      await refreshDirs();
    } catch (e) {
      setDirMsg({ type: 'error', text: String((e as Error).message) });
    }
  };

  const handleGitPull = async (id: number) => {
    setPullingId(id);
    setDirMsg(null);
    try {
      await dirGitPull(id);
      setDirMsg({ type: 'success', text: '拉取完成' });
      await refreshDirs();
    } catch (e) {
      setDirMsg({ type: 'error', text: String((e as Error).message) });
    } finally {
      setPullingId(null);
    }
  };

  const handleLoadIgnore = () => {
    setIgnoreMsg(null);
    ignoreGet().then((t) => setIgnoreContent(t || '')).catch((e) => setIgnoreMsg({ type: 'error', text: String((e as Error).message) }));
  };

  const handleSaveIgnore = async () => {
    setIgnoreMsg(null);
    try {
      await ignorePut(ignoreContent);
      setIgnoreMsg({ type: 'success', text: '已保存，热重载生效' });
    } catch (e) {
      setIgnoreMsg({ type: 'error', text: String((e as Error).message) });
    }
  };

  return (
    <main className="space-y-8">
      <header className="flex flex-wrap items-center justify-between gap-4 border-b border-zinc-200 pb-6 dark:border-zinc-800">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-zinc-900 dark:text-zinc-100">codex-mcp</h1>
          <p className="mt-0.5 text-sm text-zinc-500 dark:text-zinc-400">管理索引目录与忽略规则</p>
        </div>
        <button
          type="button"
          className="rounded-md border border-zinc-300 bg-white px-3 py-1.5 text-sm font-medium text-zinc-700 hover:bg-zinc-50 dark:border-zinc-600 dark:bg-zinc-800 dark:text-zinc-300 dark:hover:bg-zinc-700"
          onClick={openIgnoreModal}
        >
          编辑忽略规则
        </button>
      </header>

      {/* 添加目录 - 放在最上面 */}
      <section className="rounded-xl border border-zinc-200 bg-white p-6 shadow-sm dark:border-zinc-800 dark:bg-zinc-900/50">
        <h2 className="mb-4 text-lg font-semibold text-zinc-800 dark:text-zinc-200">添加目录</h2>
        <div className="flex flex-wrap items-center gap-3">
          <input
            type="text"
            placeholder="名称"
            value={addName}
            onChange={(e) => setAddName(e.target.value)}
            className="w-28 rounded-md border border-zinc-300 px-3 py-2 text-sm dark:border-zinc-600 dark:bg-zinc-800 dark:text-zinc-200"
          />
          <input
            type="text"
            placeholder="绝对路径"
            value={addPath}
            onChange={(e) => setAddPath(e.target.value)}
            className="min-w-[280px] flex-1 rounded-md border border-zinc-300 px-3 py-2 text-sm dark:border-zinc-600 dark:bg-zinc-800 dark:text-zinc-200"
          />
          <select
            value={addLang}
            onChange={(e) => setAddLang(e.target.value)}
            className="w-32 rounded-md border border-zinc-300 px-3 py-2 text-sm dark:border-zinc-600 dark:bg-zinc-800 dark:text-zinc-200"
          >
            {LANGUAGE_OPTIONS.map((opt) => (
              <option key={opt.value || '_'} value={opt.value}>{opt.label}</option>
            ))}
          </select>
          <select
            value={addRole}
            onChange={(e) => setAddRole(e.target.value)}
            className="w-28 rounded-md border border-zinc-300 px-3 py-2 text-sm dark:border-zinc-600 dark:bg-zinc-800 dark:text-zinc-200"
          >
            {ROLE_OPTIONS.map((opt) => (
              <option key={opt.value || '_'} value={opt.value}>{opt.label}</option>
            ))}
          </select>
          <button
            type="button"
            className="rounded-md bg-zinc-800 px-4 py-2 text-sm font-medium text-white hover:bg-zinc-700 dark:bg-zinc-700 dark:hover:bg-zinc-600"
            onClick={handleAdd}
          >
            添加
          </button>
        </div>
        {dirMsg && (
          <p className={`mt-3 text-sm ${dirMsg.type === 'error' ? 'text-red-600 dark:text-red-400' : 'text-green-600 dark:text-green-400'}`}>
            {dirMsg.text}
          </p>
        )}
      </section>

      <section className="rounded-xl border border-zinc-200 bg-white p-6 shadow-sm dark:border-zinc-800 dark:bg-zinc-900/50">
        <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
          <h2 className="text-lg font-semibold text-zinc-800 dark:text-zinc-200">目录列表</h2>
          {!loading && (
            <button
              type="button"
              className="rounded-md border border-zinc-300 bg-white px-3 py-1.5 text-sm font-medium text-zinc-700 hover:bg-zinc-50 dark:border-zinc-600 dark:bg-zinc-800 dark:text-zinc-300 dark:hover:bg-zinc-700"
              onClick={() => { setDirMsg(null); refreshDirs(); }}
            >
              刷新
            </button>
          )}
        </div>
        <div className="overflow-x-auto">
          {loading ? (
            <p className="py-8 text-center text-sm text-zinc-500">加载中…</p>
          ) : dirs.length === 0 ? (
            <p className="py-8 text-center text-sm text-zinc-500">暂无目录，请在上方添加</p>
          ) : (
            <table className="w-full min-w-[1280px] border-collapse text-sm">
              <thead>
                <tr className="border-b border-zinc-200 dark:border-zinc-700">
                  <th className="w-10 py-3 pr-2 text-left font-medium text-zinc-600 dark:text-zinc-400">ID</th>
                  <th className="w-24 py-3 pr-2 text-left font-medium text-zinc-600 dark:text-zinc-400">名称</th>
                  <th className="min-w-[560px] py-3 pr-2 text-left font-medium text-zinc-600 dark:text-zinc-400">路径</th>
                  <th className="w-24 py-3 pr-2 text-left font-medium text-zinc-600 dark:text-zinc-400">语言</th>
                  <th className="w-24 py-3 pr-2 text-left font-medium text-zinc-600 dark:text-zinc-400">角色</th>
                  <th className="w-14 py-3 pr-2 text-left font-medium text-zinc-600 dark:text-zinc-400">启用</th>
                  <th className="min-w-[200px] py-3 pr-2 text-left font-medium text-zinc-600 dark:text-zinc-400">Git 自动更新</th>
                  <th className="w-28 py-3 text-left font-medium text-zinc-600 dark:text-zinc-400">操作</th>
                </tr>
              </thead>
              <tbody>
                {dirs.map((d) => (
                  <tr key={d.id} className="border-b border-zinc-100 dark:border-zinc-800">
                    <td className="py-2.5 pr-2 text-zinc-500">{d.id}</td>
                    <td className="py-2.5 pr-2">{d.name}</td>
                    <td className="min-w-[560px] max-w-[720px] truncate py-2.5 pr-2 font-mono text-xs text-zinc-600 dark:text-zinc-400" title={d.path}>{d.path}</td>
                    <td className="py-2.5 pr-2 text-zinc-600 dark:text-zinc-400">{d.language || '—'}</td>
                    <td className="py-2.5 pr-2 text-zinc-600 dark:text-zinc-400">{d.role || '—'}</td>
                    <td className="py-2.5 pr-2">{d.enabled ? '是' : '否'}</td>
                    <td className="py-2.5 pr-2">
                      <div className="flex flex-wrap items-center gap-2">
                        <select
                          className="rounded border border-zinc-300 bg-white px-2 py-1 text-sm dark:border-zinc-600 dark:bg-zinc-800 dark:text-zinc-200"
                          value={d.git_auto_update_interval_sec ?? 0}
                          onChange={(e) => handleGitInterval(d.id, Number(e.target.value))}
                        >
                          {GIT_INTERVALS.map((opt) => (
                            <option key={opt.value} value={opt.value}>{opt.label}</option>
                          ))}
                        </select>
                        <span className="text-xs text-zinc-500">{formatGitLastUpdated(d.git_last_updated_at)}</span>
                        <button
                          type="button"
                          disabled={pullingId === d.id}
                          className="rounded bg-zinc-200 px-2 py-1 text-xs hover:bg-zinc-300 disabled:opacity-50 dark:bg-zinc-700 dark:hover:bg-zinc-600"
                          onClick={() => handleGitPull(d.id)}
                        >
                          {pullingId === d.id ? '拉取中…' : '手动更新'}
                        </button>
                      </div>
                    </td>
                    <td className="py-2.5">
                      <div className="flex gap-2">
                        <button
                          type="button"
                          className="rounded bg-zinc-200 px-2 py-1 text-xs hover:bg-zinc-300 dark:bg-zinc-700 dark:hover:bg-zinc-600"
                          onClick={() => handleToggle(d)}
                        >
                          {d.enabled ? '禁用' : '启用'}
                        </button>
                        <button
                          type="button"
                          className="rounded bg-red-100 px-2 py-1 text-xs text-red-700 hover:bg-red-200 dark:bg-red-900/30 dark:text-red-400 dark:hover:bg-red-900/50"
                          onClick={() => handleDelete(d.id)}
                        >
                          删除
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </section>

      {/* 忽略规则 Modal */}
      {ignoreModalOpen && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
          onClick={() => setIgnoreModalOpen(false)}
          role="dialog"
          aria-modal="true"
          aria-labelledby="ignore-modal-title"
        >
          <div
            className="w-full max-w-2xl rounded-xl border border-zinc-200 bg-white p-6 shadow-xl dark:border-zinc-700 dark:bg-zinc-900"
            onClick={(e) => e.stopPropagation()}
          >
            <h2 id="ignore-modal-title" className="mb-1 text-lg font-semibold text-zinc-800 dark:text-zinc-200">编辑忽略规则</h2>
            <p className="mb-4 text-sm text-zinc-500 dark:text-zinc-400">gitignore 格式，每行一条；保存后热重载，无需重启。</p>
            <textarea
              value={ignoreContent}
              onChange={(e) => setIgnoreContent(e.target.value)}
              placeholder="# 示例 .git node_modules *.log"
              rows={12}
              className="w-full rounded-lg border border-zinc-300 bg-white px-3 py-2 font-mono text-sm dark:border-zinc-600 dark:bg-zinc-800 dark:text-zinc-200"
            />
            <div className="mt-4 flex flex-wrap items-center gap-3">
              <button
                type="button"
                className="rounded-md border border-zinc-300 bg-white px-3 py-1.5 text-sm font-medium text-zinc-700 hover:bg-zinc-50 dark:border-zinc-600 dark:bg-zinc-800 dark:text-zinc-300 dark:hover:bg-zinc-700"
                onClick={handleLoadIgnore}
              >
                重新加载
              </button>
              <button
                type="button"
                className="rounded-md bg-zinc-800 px-3 py-1.5 text-sm font-medium text-white hover:bg-zinc-700 dark:bg-zinc-700 dark:hover:bg-zinc-600"
                onClick={handleSaveIgnore}
              >
                保存
              </button>
              {ignoreMsg && (
                <span className={`text-sm ${ignoreMsg.type === 'error' ? 'text-red-600 dark:text-red-400' : 'text-green-600 dark:text-green-400'}`}>
                  {ignoreMsg.text}
                </span>
              )}
              <button
                type="button"
                className="ml-auto rounded-md border border-zinc-300 bg-white px-3 py-1.5 text-sm font-medium text-zinc-700 hover:bg-zinc-50 dark:border-zinc-600 dark:bg-zinc-800 dark:text-zinc-300 dark:hover:bg-zinc-700"
                onClick={() => setIgnoreModalOpen(false)}
              >
                关闭
              </button>
            </div>
          </div>
        </div>
      )}
    </main>
  );
}
