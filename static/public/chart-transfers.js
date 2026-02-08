const elem = document.getElementById('transfer-chart');
const myChart = echarts.init(elem, null, {});
window.addEventListener('resize', function () {
    myChart.resize();
});

async function load() {
    const url = `/transfers/chart/data${elem.getAttribute("x-filter") || window.location.search}`
    const raw = await fetch(url);
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