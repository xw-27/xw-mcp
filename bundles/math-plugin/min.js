module.exports = {
    type: 'tool',
    name: 'min',
    description: '返回一组数字中的最小值',
    inputSchema: {
        type: 'object',
        properties: {
            values: {
                type: 'array',
                items: {
                    type: 'number',
                    description: '数字'
                },
                description: '数字数组'
            }
        },
        required: ['values']
    },
    execute: function(params) {
        const result = Math.min.apply(null, params.values);
        log.info(`min called: min([${params.values}]) = ${result}`);
        return { result: result };
    }
}