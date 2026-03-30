(function() {
    function getThemeColor(cssVar, fallback) {
        try {
            var v = getComputedStyle(document.documentElement).getPropertyValue(cssVar).trim();
            return v || fallback;
        } catch (_) {
            return fallback;
        }
    }

    function getThemeOpts() {
        var baseContent = getThemeColor('--color-base-content', '#333');
        var baseBg = getThemeColor('--color-base-100', '#fff');
        var tooltipBorder = getThemeColor('--color-base-300', '#ccc');
        return {
            baseContent: baseContent,
            baseBg: baseBg,
            tooltipBorder: tooltipBorder,
            textStyle: { color: baseContent, textBorderWidth: 0 },
            axisLabel: { color: baseContent, textBorderWidth: 0 }
        };
    }

    function formatThousands(val) {
        var n = Math.round(val);
        var neg = n < 0;
        var s = Math.abs(n).toString();
        var pre = s.length % 3 || 3;
        var out = s.slice(0, pre);
        for (var i = pre; i < s.length; i += 3) {
            out += ',' + s.slice(i, i + 3);
        }
        return neg ? '-' + out : out;
    }

    function lightenColor(hex) {
        if (!hex || hex.length < 7) return 'rgba(100, 100, 100, 0.2)';
        var r = parseInt(hex.slice(1, 3), 16);
        var g = parseInt(hex.slice(3, 5), 16);
        var b = parseInt(hex.slice(5, 7), 16);
        return 'rgba(' + r + ', ' + g + ', ' + b + ', 0.2)';
    }

    var dom = document.getElementById('dashboard-forecast-chart');
    if (!dom) return;

    var chart = echarts.init(dom);
    var series = {};
    var legendData = [];
    var allMarklines = [];
    var statusEl = document.getElementById('forecast-status');
    var durationSelect = document.getElementById('forecast-duration');

    function getMaxDate() {
        var years = parseInt(durationSelect ? durationSelect.value : '10', 10);
        if (years === 0) return null; // max = no limit
        var d = new Date();
        d.setFullYear(d.getFullYear() + years);
        return d.getTime();
    }

    function filterData(data) {
        var maxDate = getMaxDate();
        if (!maxDate) return data;
        return data.filter(function(point) { return point[0] <= maxDate; });
    }

    function showLoading() {
        if (statusEl) statusEl.classList.remove('hidden');
    }

    function hideLoading() {
        if (statusEl) statusEl.classList.add('hidden');
    }

    function addEntity(e) {
        var color = e.color || '#666';
        series[e.id] = {
            id: e.id,
            name: e.name,
            type: 'line',
            data: [],
            showSymbol: false,
            smooth: false,
            group: e.name,
            lineStyle: { color: color },
            itemStyle: { color: color }
        };
        series[e.id + '_min'] = {
            id: e.id + '_min',
            name: e.name + ' min',
            type: 'line',
            data: [],
            lineStyle: { opacity: 0 },
            stack: e.id + '-confidence-band',
            symbol: 'none',
            showSymbol: false,
            smooth: false,
            group: e.name,
            tooltip: { show: false },
            label: { show: false }
        };
        series[e.id + '_max'] = {
            id: e.id + '_max',
            name: e.name + ' max',
            type: 'line',
            data: [],
            lineStyle: { opacity: 0 },
            stack: e.id + '-confidence-band',
            showSymbol: false,
            smooth: false,
            group: e.name,
            areaStyle: { color: lightenColor(color) },
            tooltip: { show: false },
            label: { show: false }
        };
        legendData.push({ name: e.name });
    }

    function addSnapshotFromEntity(s, entityId) {
        if (!series[entityId]) return;
        series[entityId].data.push([s.day, s.balance]);
        series[entityId + '_min'].data.push([s.day, s.lowerBound]);
        series[entityId + '_max'].data.push([s.day, s.upperBound - s.lowerBound]);
    }

    function addSnapshotFromSSE(s) {
        var id = s.AccountTypeID;
        if (!series[id]) return;
        series[id].data.push([s.Date, s.Median]);
        series[id + '_min'].data.push([s.Date, s.LowerBound]);
        series[id + '_max'].data.push([s.Date, s.UpperBound - s.LowerBound]);
    }

    function buildFilteredSeries() {
        var maxDate = getMaxDate();
        var filtered = Object.keys(series).map(function(key) {
            var s = series[key];
            var copy = {};
            for (var k in s) { copy[k] = s[k]; }
            copy.data = maxDate ? s.data.filter(function(p) { return p[0] <= maxDate; }) : s.data;
            return copy;
        });
        var filteredMarklines = allMarklines.filter(function(m) {
            if (!maxDate) return true;
            return m._date <= maxDate;
        }).map(function(m) {
            var copy = {};
            for (var k in m) { if (k !== '_date') copy[k] = m[k]; }
            return copy;
        });
        return filtered.concat(filteredMarklines);
    }

    function updateChart() {
        var theme = getThemeOpts();
        chart.setOption({
            backgroundColor: theme.baseBg,
            tooltip: {
                trigger: 'axis',
                backgroundColor: theme.baseBg,
                borderColor: theme.tooltipBorder,
                extraCssText: 'box-shadow: 0 2px 4px rgba(0,0,0,0.15);',
                formatter: function(params) {
                    var items = params.filter(function(p) {
                        return p.seriesName.indexOf(' min') === -1 &&
                               p.seriesName.indexOf(' max') === -1 &&
                               p.value && p.value[1] !== 0;
                    });
                    if (items.length === 0) return '';
                    var date = new Date(items[0].value[0]);
                    var header = date.getFullYear() + '-' + String(date.getMonth()+1).padStart(2,'0') + '-' + String(date.getDate()).padStart(2,'0');
                    var total = items.reduce(function(sum, p) { return sum + p.value[1]; }, 0);
                    var lines = items.map(function(p) {
                        return '<div style="display:flex;justify-content:space-between;gap:16px">' +
                            '<span>' + p.marker + ' ' + p.seriesName + '</span>' +
                            '<span style="font-weight:500">' + formatThousands(p.value[1]) + '</span></div>';
                    });
                    var totalLine = '<div style="display:flex;justify-content:space-between;gap:16px;border-top:1px solid ' + theme.tooltipBorder + ';margin-top:4px;padding-top:4px">' +
                        '<span>Total</span><span style="font-weight:700">' + formatThousands(total) + '</span></div>';
                    return header + '<br/>' + lines.join('') + totalLine;
                }
            },
            legend: {
                data: legendData,
                type: 'scroll',
                bottom: 0,
                textStyle: theme.textStyle
            },
            grid: {
                left: '3%',
                right: '4%',
                bottom: '15%',
                containLabel: true
            },
            xAxis: {
                type: 'time',
                max: getMaxDate() || undefined,
                axisLabel: theme.axisLabel
            },
            yAxis: {
                type: 'value',
                axisLabel: theme.axisLabel
            },
            series: buildFilteredSeries()
        });
    }

    var evtSource = new EventSource('/dashboard/forecast/stream');

    evtSource.addEventListener('setup', function(event) {
        var data = JSON.parse(event.data);
        (data.entities || []).forEach(function(e) {
            addEntity(e);
            (e.snapshots || []).forEach(function(s) {
                addSnapshotFromEntity(s, e.id);
            });
        });

        var themeText = getThemeColor('--color-base-content', '#333');
        allMarklines = (data.marklines || []).map(function(m, idx) {
            return {
                _date: m.date,
                name: m.name,
                type: 'line',
                markLine: {
                    symbol: ['none', 'none'],
                    data: [{
                        xAxis: new Date(m.date),
                        lineStyle: { color: m.color || themeText, type: 'dashed' },
                        label: {
                            offset: [0, idx % 2 !== 0 ? 0 : -15],
                            formatter: m.name,
                            color: m.color || themeText,
                            textBorderWidth: 0
                        }
                    }]
                }
            };
        });

        updateChart();
    });

    evtSource.addEventListener('snapshot', function(event) {
        var s = JSON.parse(event.data);
        addSnapshotFromSSE(s);
        updateChart();
    });

    evtSource.addEventListener('status', function(event) {
        var data = JSON.parse(event.data);
        if (data.status === 'running') {
            showLoading();
        } else {
            hideLoading();
        }
    });

    evtSource.addEventListener('close', function() {
        evtSource.close();
        hideLoading();
    });

    chart.on('legendselectchanged', function(params) {
        Object.values(series).forEach(function(v) {
            chart.dispatchAction({
                type: params.selected[v.group] ? 'legendSelect' : 'legendUnSelect',
                name: v.name
            });
        });
    });

    if (durationSelect) {
        durationSelect.addEventListener('change', function() { updateChart(); });
    }

    window.addEventListener('resize', function() { chart.resize(); });
    window.addEventListener('themechange', function() { updateChart(); chart.resize(); });
})();
