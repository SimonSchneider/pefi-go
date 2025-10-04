const colors = [
    `#D32F2F`,
    `#F57C00`,
    `#FFB300`,
    `#FBC02D`,
    `#388E3C`,
    `#00897B`,
    `#00ACC1`,
    `#1976D2`,
    `#303F9F`,
    `#7B1FA2`,
    `#C2185B`,
    `#795548`,
    `#616161`,
];

const myChart = echarts.init(document.getElementById('main'), null, {
    // renderer: 'svg'
});
window.addEventListener('resize', function () {
    myChart.resize();
});

const series = {};

function getColor(idx) {
    return colors[idx % colors.length];
}

function lightenColor(color) {
    const [r, g, b] = color.match(/\w\w/g).map(c => parseInt(c, 16));
    return `rgba(${r}, ${g}, ${b}, 0.2)`;
}

async function load() {
    const raw = await fetch(`data${window.location.search}`);
    const data = await raw.json();

    myChart.setOption({
        series: [{
            type: 'sankey',
            layout: 'none',
            emphasis: {
                focus: 'adjacency'
            },
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