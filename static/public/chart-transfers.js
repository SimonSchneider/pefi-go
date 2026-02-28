function getThemeColor(cssVar, fallback) {
    try {
        const v = getComputedStyle(document.documentElement).getPropertyValue(cssVar).trim();
        return v || fallback;
    } catch (_) {
        return fallback;
    }
}

const elem = document.getElementById('transfer-chart');
const myChart = echarts.init(elem, null, {});
window.addEventListener('resize', function () {
    myChart.resize();
});

function getTransferThemeOpts() {
    const textColor = getThemeColor('--color-base-content', '#333');
    const bgColor = getThemeColor('--color-base-100', 'transparent');
    const borderColor = getThemeColor('--color-base-300', '#ccc');
    return {
        backgroundColor: bgColor,
        tooltip: {
            backgroundColor: bgColor,
            borderColor: borderColor,
            textStyle: { color: textColor, textBorderWidth: 0 },
        },
        series: [{
            label: {
                color: textColor,
                textBorderWidth: 0,
                textBorderColor: 'transparent',
            },
        }],
    };
}

function applyTransferChartTheme() {
    myChart.setOption(getTransferThemeOpts());
    myChart.resize();
}

window.addEventListener('themechange', applyTransferChartTheme);

async function load() {
    const url = `/transfers/chart/data${elem.getAttribute("x-filter") || window.location.search}`
    const raw = await fetch(url);
    const data = await raw.json();

    const themeOpts = getTransferThemeOpts();
    myChart.setOption({
        backgroundColor: themeOpts.backgroundColor,
        tooltip: themeOpts.tooltip,
        series: [{
            type: 'sankey',
            layout: 'none',
            emphasis: {
                focus: 'adjacency'
            },
            label: themeOpts.series[0].label,
            data: data.data,
            links: data.links,
            lineStyle: {
                color: 'target',
                curveness: 0.5
            }
        }]
    });
}

load();