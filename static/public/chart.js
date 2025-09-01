const colors = [
    [`#F4A6A6`, `#D32F2F`],
    [`#F7C59F`, `#F57C00`],
    [`#FBE8A6`, `#FFB300`],
    [`#FFF7AE`, `#FBC02D`],
    [`#B2D8B2`, `#388E3C`],
    [`#A8DAD3`, `#00897B`],
    [`#B2EBF2`, `#00ACC1`],
    [`#A6C8FF`, `#1976D2`],
    [`#C5CAE9`, `#303F9F`],
    [`#D1C4E9`, `#7B1FA2`],
    [`#F8BBD0`, `#C2185B`],
    [`#D7CCC8`, `#795548`],
    [`#E0E0E0`, `#616161`],
];

const myChart = echarts.init(document.getElementById('main'), null, {
    // renderer: 'svg'
});
window.addEventListener('resize', function () {
    myChart.resize();
});
const batchInterval = 100;

const series = {};

function getColor(idx) {
    return colors[idx % colors.length][1];
}

function lightenColor(color) {
    const [r, g, b] = color.match(/\w\w/g).map(c => parseInt(c, 16));
    return `rgba(${r}, ${g}, ${b}, 0.3)`;
}

myChart.setOption({
    legend: {
        data: [],
        selectedMode: 'multiple',
        type: 'scroll',
        orient: 'vertical',
        top: 20,
        right: 10,
    },
    grid: {
        containLabel: true
    },
    animationDurationUpdate: batchInterval,
    tooltip: {
        order: 'valueDesc',
        trigger: 'axis',
        valueFormatter: (value) => `${value.toLocaleString('en-us', { maximumFractionDigits: 0 })} kr`,
    },
    xAxis: {
        type: 'time',
        name: 'Date',
        nameLocation: 'middle',
        nameGap: 30,
    },
    yAxis: {
        type: 'value',
        name: 'Balance',
        nameLocation: 'middle',
        nameGap: 100,
        axisLabel: { formatter: '{value} kr' },
    },
    dataZoom: [
        {
            type: 'inside',   // Enables zooming with mouse wheel and drag
            xAxisIndex: 0     // Applies to first xAxis
        },
        {
            type: 'slider',   // Optional: visible slider below the chart
            xAxisIndex: 0
        }
    ],
    series: []
});

const addPointToSeries = (seriesName, day, balance) => (series[seriesName].data || []).push([day, balance]);
const addDataPoint = (dp) => {
    addPointToSeries(dp.id, dp.day, dp.balance);
    addPointToSeries(`${dp.id}_min`, dp.day, dp.lowerBound);
    addPointToSeries(`${dp.id}_max`, dp.day, dp.upperBound - dp.lowerBound);
}
const runUpdateBatch = () => myChart.setOption({ series: Object.values(series) });
const intervalUpdate = setInterval(runUpdateBatch, batchInterval);

let idx = 0;

const addSeries = (data) => {
    const color = data.color || getColor(idx);
    series[data.id] = {
        id: data.id,
        name: data.name,
        type: 'line',
        data: [],
        showSymbol: false,
        smooth: false,
        group: data.name,
        lineStyle: {
            color,
        },
        itemStyle: {
            color,
        },
    }
    series[`${data.id}_min`] = {
        id: `${data.id}_min`,
        name: `${data.name} min`,
        type: 'line',
        data: [],
        lineStyle: { opacity: 0 },
        stack: `${data.id}-confidence-band`,
        symbol: 'none',
        showSymbol: false,
        smooth: false,
        group: data.name,
        tooltip: {
            show: false,
        },
        label: {
            show: false,
        },
    }
    series[`${data.id}_max`] = {
        id: `${data.id}_max`,
        name: `${data.name} max`,
        type: 'line',
        data: [],
        lineStyle: { opacity: 0 },
        stack: `${data.id}-confidence-band`,
        showSymbol: false,
        smooth: false,
        group: data.name,
        areaStyle: {
            color: lightenColor(color)
        },
        tooltip: {
            show: false,
        },
        label: {
            show: false,
        }
    }
    idx++;
}

myChart.on('legendselectchanged', function (params) {
    Object.values(series).forEach(v => {
        myChart.dispatchAction({
            type: params.selected[v.group] ? 'legendSelect' : 'legendUnSelect',
            name: v.name,
        });
    })
});

const evtSource = new EventSource(`/chart/stream${window.location.search}`);
evtSource.addEventListener('setup', (event) => {
    const data = JSON.parse(event.data);
    data.entities.forEach(e => {
        addSeries(e);
        e.snapshots.forEach(s => addDataPoint(s));
    })
    console.log(data.marklines);
    const today = new Date();
    const today2 = new Date();
    today2.setDate(today2.getUTCDate() + 1000);
    myChart.setOption({
        legend: {
            data: data.entities.map(e => ({ name: e.name })),
        },
        xAxis: { max: data.max },
        series: Object.values(series).concat([
            {
                id: 'today',
                name: 'Today',
                type: 'line',
                markLine: {
                    symbol: ['none', 'none'],
                    data: [
                        {
                            xAxis: today,
                            lineStyle: {
                                color: '#ff0000',
                                type: 'dashed',
                            },
                            label: {
                                formatter: 'Today',
                                color: '#ff0000',
                            }
                        }
                    ]
                },
            },
            ...data.marklines.map((m, idx) => ({
                id: m.name,
                name: m.name,
                type: 'line',
                markLine: {
                    symbol: ['none', 'none'],
                    data: [
                        {
                            xAxis: new Date(m.date),
                            lineStyle: {
                                color: '#ff0000',
                                type: 'dashed',
                            },
                            label: {
                                offset: [0, idx % 2 !== 0 ? 0 : -15],
                                formatter: m.name,
                                color: '#ff0000',
                            }
                        }
                    ]
                }
            })),
        ]),
    })
});

evtSource.addEventListener('balanceSnapshot', (event) => addDataPoint(JSON.parse(event.data)));
evtSource.addEventListener('close', (event) => {
    evtSource.close();
    clearInterval(intervalUpdate);
    runUpdateBatch();
});
