module.exports = {
    type: 'tool',
    name: 'max',
    description: '返回一组数字中的最大值',
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
        const result = Math.max.apply(null, params.values);
        log.info(`max called: max([${params.values}]) = ${result}`);
        return { result: result };
    }
}