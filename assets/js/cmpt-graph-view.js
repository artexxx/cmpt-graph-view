import params from '@params';

class CmptGraphView {
    static instances = [];
    static linkmap = null;

    static norm(p) {
        if (!p) return '/';
        p = String(p).trim().split('#')[0].split('?')[0];
        if (!p.startsWith('/')) p = '/' + p;
        if (p.length > 1 && p.endsWith('/')) p = p.slice(0, -1);
        return p || '/';
    }

    static isMobileLike(bp) {
        const mm = window.matchMedia ? window.matchMedia.bind(window) : null;
        return (window.innerWidth <= bp) ||
            (mm && mm('(hover: none)').matches) ||
            (mm && mm('(pointer: coarse)').matches);
    }

    static getIsDark() {
        if (typeof window.fixit?.isDark === 'boolean') return window.fixit.isDark;
        const dt = document.documentElement.getAttribute('data-theme');
        if (dt) return dt === 'dark';
        return document.documentElement.classList.contains('dark');
    }

    static setAllTheme(isDark) {
        CmptGraphView.instances.forEach((inst) => inst.setTheme(isDark));
    }

    static initThemeEvents() {
        const apply = (isDark) => CmptGraphView.setAllTheme(!!isDark);

        apply(CmptGraphView.getIsDark());

        if (typeof window.fixit?.switchThemeEventSet === 'object' && window.fixit?.switchThemeEventSet?.add) {
            window.fixit.switchThemeEventSet.add((isDark) => apply(isDark));
            return;
        }

        const mo = new MutationObserver(() => apply(CmptGraphView.getIsDark()));
        mo.observe(document.documentElement, { attributes: true, attributeFilter: ['class', 'data-theme', 'style'] });
    }

    static readLinkmapOnce() {
        const el = document.getElementById('gv-linkmap-json');
        if (!el) return null;
        try {
            const raw = JSON.parse(el.textContent || '{}');
            return (typeof raw === 'string') ? JSON.parse(raw) : raw;
        } catch {
            return null;
        }
    }

    static boot() {
        if (!params.enable) return;

        const roots = document.querySelectorAll('.gv[data-gv]');
        if (!roots.length) return;

        const bp = Number(params.mobileBp ?? 900);
        if (CmptGraphView.isMobileLike(bp)) return;

        CmptGraphView.linkmap = CmptGraphView.readLinkmapOnce();
        if (!CmptGraphView.linkmap) return;

        roots.forEach((root) => {
            try {
                const inst = new CmptGraphView(root, CmptGraphView.linkmap);
                inst.bind();
                CmptGraphView.instances.push(inst);
            } catch {}
        });

        CmptGraphView.initThemeEvents();
    }

    constructor(root, linkmap) {
        this.root = root;
        this.id = root.getAttribute('data-gv-id') || '';

        this.canvas = root.querySelector(`#gv-canvas-${this.id}`);
        this.backlinksBox = root.querySelector(`#gv-backlinks-${this.id}`);
        this.dialog = root.querySelector(`#gv-dialog-${this.id}`);
        this.canvasFull = root.querySelector(`#gv-canvas-full-${this.id}`);

        this.btnBacklinks = root.querySelector('[data-gv-action="backlinks"]');
        this.btnFull = root.querySelector('[data-gv-action="full"]');
        this.btnClose = root.querySelector('[data-gv-action="close"]');

        this.index = (linkmap && linkmap.index) || { links: {}, backlinks: {} };
        this.contentMap = (linkmap && linkmap.content) || {};

        this.curPage = this.getCurPage();

        this.height = Number(root.getAttribute('data-gv-height') || 400);
        this.fullHeight = Number(root.getAttribute('data-gv-full-height') || 720);
        this.showLabelsInFull = (root.getAttribute('data-gv-show-labels-full') === '1');

        this.stateBacklinks = false;

        this.graphMain = null;
        this.graphFull = null;
    }

    getCurPage() {
        return CmptGraphView.norm(window.location.pathname || '/');
    }

    titleOf(path) {
        const o = (this.contentMap && this.contentMap[path]) || null;
        return (o && o.title) ? o.title : path;
    }

    dedupeEdges(edges) {
        const set = new Set();
        const out = [];
        for (const e of edges) {
            const k = `${e.source}->${e.target}`;
            if (set.has(k)) continue;
            set.add(k);
            out.push(e);
        }
        return out;
    }

    buildOneHop() {
        const cur = this.curPage;
        const neigh = new Set([cur]);

        const outgoing = (this.index.links && this.index.links[cur]) || [];
        const incoming = (this.index.backlinks && this.index.backlinks[cur]) || [];

        outgoing.forEach(l => neigh.add(CmptGraphView.norm(l.target)));
        incoming.forEach(l => neigh.add(CmptGraphView.norm(l.source)));

        const nodes = [];
        neigh.forEach(id => nodes.push({ id, label: this.titleOf(id), url: id }));

        const edges = [];
        outgoing.forEach(l => { const t = CmptGraphView.norm(l.target); if (neigh.has(t)) edges.push({ source: cur, target: t }); });
        incoming.forEach(l => { const s = CmptGraphView.norm(l.source); if (neigh.has(s)) edges.push({ source: s, target: cur }); });

        neigh.forEach(n => {
            const out = (this.index.links && this.index.links[n]) || [];
            out.forEach(l => {
                const t = CmptGraphView.norm(l.target);
                if (neigh.has(t)) edges.push({ source: n, target: t });
            });
        });

        return { nodes, edges: this.dedupeEdges(edges), showLabels: true };
    }

    buildFull() {
        const nodesMap = new Map();
        for (const k in (this.contentMap || {})) {
            nodesMap.set(k, { id: k, label: this.titleOf(k), url: k });
        }

        const edges = [];
        const allLinks = (this.index && this.index.links) || {};
        for (const src in allLinks) {
            const arr = allLinks[src] || [];
            for (const l of arr) {
                const s = CmptGraphView.norm(l.source);
                const t = CmptGraphView.norm(l.target);
                if (!nodesMap.has(s)) nodesMap.set(s, { id: s, label: this.titleOf(s), url: s });
                if (!nodesMap.has(t)) nodesMap.set(t, { id: t, label: this.titleOf(t), url: t });
                edges.push({ source: s, target: t });
            }
        }

        if (!nodesMap.has(this.curPage)) {
            nodesMap.set(this.curPage, { id: this.curPage, label: this.titleOf(this.curPage), url: this.curPage });
        }

        return { nodes: Array.from(nodesMap.values()), edges: this.dedupeEdges(edges), showLabels: !!this.showLabelsInFull };
    }

    renderBacklinksList() {
        const incoming = (this.index.backlinks && this.index.backlinks[this.curPage]) || [];
        const seen = new Set();
        const items = [];

        for (const l of incoming) {
            const src = CmptGraphView.norm(l.source);
            if (seen.has(src)) continue;
            seen.add(src);
            items.push({ path: src, title: this.titleOf(src) });
        }

        this.backlinksBox.innerHTML = '';

        const title = document.createElement('h2');
        title.className = "backlinks"
        title.textContent = "Backlinks"
        this.backlinksBox.appendChild(title);

        if (!items.length) { this.backlinksBox.textContent = 'No backlinks.'; return; }

        const ul = document.createElement('ul');
        for (const it of items) {
            const li = document.createElement('li');
            const a = document.createElement('a');
            a.href = it.path;
            a.textContent = it.title;
            li.appendChild(a);
            ul.appendChild(li);
        }
        this.backlinksBox.appendChild(ul);
    }

    readColors() {
        const s = getComputedStyle(this.root);
        const get = (k, fallback) => (s.getPropertyValue(k) || '').trim() || fallback;

        return {
            activeNode: get('--gv-node-active', '#4A90E2'),
            node: get('--gv-node', '#8FB3D9'),
            link: get('--gv-link', 'rgba(127,127,127,.35)'),
            linkActive: get('--gv-link-active', '#4A90E2'),
            text: get('--gv-text', 'currentColor')
        };
    }

    setTheme(isDark) {
        this.root.classList.toggle('gv--dark', !!isDark);
        this.applyThemeGraph(this.graphMain);
        this.applyThemeGraph(this.graphFull);
    }

    makeGraph(host) {
        return {
            host,
            svg: null,
            root: null,
            zoom: null,
            sim: null,
            link: null,
            node: null,
            label: null,
            adj: new Map(),
            ro: null,
            w: 0,
            h: 0,
            showLabels: true
        };
    }

    clearGraph(g) {
        if (!g || !g.host) return;
        g.host.innerHTML = '';
        if (g.sim) g.sim.stop();
        if (g.ro) g.ro.disconnect();
        g.svg = g.root = g.zoom = g.sim = g.link = g.node = g.label = null;
        g.adj = new Map();
    }

    applyThemeGraph(g) {
        if (!g || !g.svg) return;
        const c = this.readColors();
        if (g.link) g.link.attr('stroke', c.link);
        if (g.node) g.node.attr('fill', d => d.id === this.curPage ? c.activeNode : c.node);
        if (g.label) g.label.attr('fill', c.text);
    }

    setupZoom(g, d3) {
        const clamp = (v, a, b) => Math.max(a, Math.min(b, v));
        const zoomMin = Number(params.zoomMin ?? 0.25);
        const zoomMax = Number(params.zoomMax ?? 6);
        const wheelK = Number(params.zoomWheel ?? 0.002);

        const wheelDelta = (event) => {
            const e = event && event.sourceEvent ? event.sourceEvent : event;
            const dy = e && typeof e.deltaY === 'number' ? e.deltaY : 0;
            return clamp(-dy * wheelK, -0.22, 0.22);
        };

        g.zoom = d3.zoom()
            .scaleExtent([zoomMin, zoomMax])
            .wheelDelta(wheelDelta)
            .on('zoom', (event) => g.root.attr('transform', event.transform));

        g.svg.call(g.zoom);
        g.svg.on('wheel', (event) => { event.preventDefault(); }, { passive: false });
        g.svg.call(g.zoom.transform, d3.zoomIdentity);
    }

    setupResize(g) {
        g.ro = new ResizeObserver(() => this.onResize(g));
        g.ro.observe(g.host);
    }

    onResize(g) {
        if (!g || !g.svg || !g.host) return;
        const w = g.host.clientWidth || 600;
        const h = g.host.clientHeight || 400;
        if (w === g.w && h === g.h) return;

        g.w = w; g.h = h;
        g.svg.attr('width', w).attr('height', h).attr('viewBox', [0, 0, w, h]);

        if (g.sim) {
            const fx = g.sim.force('x');
            const fy = g.sim.force('y');
            const fc = g.sim.force('center');
            if (fx) fx.x(w / 2);
            if (fy) fy.y(h / 2);
            if (fc) fc.x(w / 2).y(h / 2);
            g.sim.alpha(0.6).restart();
        }
    }

    renderGraph(g, d3, data, tuning) {
        this.clearGraph(g);
        if (!g || !g.host) return;

        g.host.style.height = tuning.heightPx + 'px';

        g.w = g.host.clientWidth || 600;
        g.h = g.host.clientHeight || 400;

        const c = this.readColors();

        const nodes = data.nodes.map(n => ({ ...n }));
        const edges = data.edges.map(e => ({ source: e.source, target: e.target }));
        g.showLabels = !!data.showLabels;

        const addAdj = (a, b) => {
            if (!g.adj.has(a)) g.adj.set(a, new Set());
            g.adj.get(a).add(b);
        };
        edges.forEach(e => { addAdj(e.source, e.target); addAdj(e.target, e.source); });

        g.svg = d3.select(g.host)
            .append('svg')
            .attr('width', g.w)
            .attr('height', g.h)
            .attr('viewBox', [0, 0, g.w, g.h])
            .style('max-width', '100%')
            .style('height', 'auto');

        g.root = g.svg.append('g');

        this.setupZoom(g, d3);
        this.setupResize(g);

        const pad = Number(params.pad ?? 14);
        const clampNode = (d) => {
            d.x = Math.max(pad, Math.min(g.w - pad, d.x));
            d.y = Math.max(pad, Math.min(g.h - pad, d.y));
        };

        g.sim = d3.forceSimulation(nodes)
            .force('link', d3.forceLink(edges).id(d => d.id).distance(tuning.linkDist))
            .force('charge', d3.forceManyBody().strength(tuning.charge))
            .force('center', d3.forceCenter(g.w / 2, g.h / 2))
            .force('x', d3.forceX(g.w / 2).strength(tuning.pull))
            .force('y', d3.forceY(g.h / 2).strength(tuning.pull))
            .force('collision', d3.forceCollide().radius(d => d.id === this.curPage ? tuning.collideR + 6 : tuning.collideR));

        g.link = g.root.append('g')
            .attr('stroke', c.link)
            .attr('stroke-opacity', 0.7)
            .selectAll('line')
            .data(edges)
            .join('line')
            .attr('stroke-width', 1.5);

        g.node = g.root.append('g')
            .attr('stroke', '#fff')
            .attr('stroke-width', 1.5)
            .selectAll('circle')
            .data(nodes)
            .join('circle')
            .attr('r', d => d.id === this.curPage ? 10 : 6)
            .attr('fill', d => d.id === this.curPage ? c.activeNode : c.node)
            .style('cursor', 'pointer')
            .on('click', (_, d) => { if (d.url) window.location.href = d.url; })
            .on('mouseenter', (_, d) => this.hoverOn(g, d))
            .on('mouseleave', () => this.hoverOff(g))
            .call(d3.drag()
                .on('start', (event, d) => { if (!event.active) g.sim.alphaTarget(0.3).restart(); d.fx = d.x; d.fy = d.y; })
                .on('drag', (event, d) => { d.fx = event.x; d.fy = event.y; })
                .on('end', (event, d) => { if (!event.active) g.sim.alphaTarget(0); d.fx = null; d.fy = null; })
            );

        g.label = g.root.append('g')
            .selectAll('text')
            .data(nodes)
            .join('text')
            .text(d => d.label)
            .attr('font-size', 12)
            .attr('dx', 12)
            .attr('dy', 4)
            .attr('fill', c.text)
            .style('pointer-events', 'none')
            .style('opacity', g.showLabels ? 1 : 0);

        g.sim.on('tick', () => {
            nodes.forEach(clampNode);

            g.link
                .attr('x1', d => d.source.x)
                .attr('y1', d => d.source.y)
                .attr('x2', d => d.target.x)
                .attr('y2', d => d.target.y);

            g.node
                .attr('cx', d => d.x)
                .attr('cy', d => d.y);

            g.label
                .attr('x', d => d.x)
                .attr('y', d => d.y);
        });

        g.sim.alpha(1).restart();
    }

    hoverOn(g, d) {
        if (!g || !g.link || !g.node || !g.label) return;

        const c = this.readColors();
        const set = g.adj.get(d.id) || new Set();
        const inSet = (id) => id === d.id || set.has(id);

        g.link
            .attr('stroke', e => {
                const s = e.source.id ? e.source.id : e.source;
                const t = e.target.id ? e.target.id : e.target;
                return (s === d.id || t === d.id) ? c.linkActive : c.link;
            })
            .attr('stroke-opacity', e => {
                const s = e.source.id ? e.source.id : e.source;
                const t = e.target.id ? e.target.id : e.target;
                return (s === d.id || t === d.id) ? 0.95 : 0.2;
            })
            .attr('stroke-width', e => {
                const s = e.source.id ? e.source.id : e.source;
                const t = e.target.id ? e.target.id : e.target;
                return (s === d.id || t === d.id) ? 2.5 : 1.0;
            });

        g.node.attr('opacity', n => inSet(n.id) ? 1 : 0.35);

        g.label
            .attr('font-size', n => n.id === d.id ? 16 : 12)
            .style('opacity', n => g.showLabels ? 1 : (n.id === d.id ? 1 : 0));
    }

    hoverOff(g) {
        if (!g || !g.link || !g.node || !g.label) return;

        const c = this.readColors();

        g.link
            .attr('stroke', c.link)
            .attr('stroke-opacity', 0.7)
            .attr('stroke-width', 1.5);

        g.node.attr('opacity', 1);

        g.label
            .attr('font-size', 12)
            .style('opacity', g.showLabels ? 1 : 0);
    }

    openDialog() {
        if (!this.dialog) return;
        if (typeof this.dialog.showModal === 'function') this.dialog.showModal();
        else this.dialog.setAttribute('open', 'open');
    }

    closeDialog() {
        if (!this.dialog) return;
        if (typeof this.dialog.close === 'function') this.dialog.close();
        else this.dialog.removeAttribute('open');
    }

    bind() {
        if (!this.canvas || !this.backlinksBox) return;
        if (!window.d3) return;

        this.root.style.setProperty('--gv-height', this.height + 'px');
        this.root.style.setProperty('--gv-full-height', this.fullHeight + 'px');

        this.graphMain = this.makeGraph(this.canvas);
        this.graphFull = this.makeGraph(this.canvasFull);

        this.renderBacklinksList();
        this.backlinksBox.hidden = true;

        this.btnBacklinks?.addEventListener('click', () => {
            this.stateBacklinks = this.btnBacklinks.getAttribute('aria-pressed') !== 'true';
            this.btnBacklinks.setAttribute('aria-pressed', this.stateBacklinks ? 'true' : 'false');
            this.backlinksBox.hidden = !this.stateBacklinks;
        });

        this.btnFull?.addEventListener('click', () => {
            this.openDialog();
            requestAnimationFrame(() => {
                const data = this.buildFull();
                this.renderGraph(this.graphFull, window.d3, data, {
                    heightPx: this.fullHeight,
                    linkDist: Number(params.fullLinkDist ?? 72),
                    charge: Number(params.fullCharge ?? -170),
                    pull: Number(params.fullPull ?? 0.10),
                    collideR: Number(params.fullCollideR ?? 14),
                });
                this.applyThemeGraph(this.graphFull);
            });
        });

        this.btnClose?.addEventListener('click', () => this.closeDialog());

        this.dialog?.addEventListener('click', (e) => {
            if (e.target === this.dialog) this.closeDialog();
        });

        this.dialog?.addEventListener('close', () => {
            this.clearGraph(this.graphFull);
        });

        const data = this.buildOneHop();
        this.renderGraph(this.graphMain, window.d3, data, {
            heightPx: this.height,
            linkDist: Number(params.oneHopLinkDist ?? 110),
            charge: Number(params.oneHopCharge ?? -260),
            pull: Number(params.oneHopPull ?? 0.06),
            collideR: Number(params.oneHopCollideR ?? 22),
        });

        this.setTheme(CmptGraphView.getIsDark());
    }
}

if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => CmptGraphView.boot(), { once: true });
} else {
    CmptGraphView.boot();
}
