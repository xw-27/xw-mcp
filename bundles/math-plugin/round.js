module.exports = {
    type: 'tool',
    name: 'round',
    description: '对数字进行四舍五入',
    inputSchema: {
        type: 'object',
        properties: {
            value: {
                type: 'number',
                description: '待四舍五入的数字'
            },
            decimals: {
                type: 'number',
                description: '保留小数位数，默认为0'
            }
        },
        required: ['value']
    },
    execute: function(params) {
        const multiplier = Math.pow(10, params.decimals || 0);
        const result = Math.round(params.value * multiplier) / multiplier;
        log.info(`round called: round(${params.value}, ${params.decimals || 0}) = ${result}`);
        return { result: result };
    }
}