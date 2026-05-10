module.exports = {
    type: 'tool',
    name: 'sqrt',
    description: '计算平方根',
    inputSchema: {
        type: 'object',
        properties: {
            value: {
                type: 'number',
                description: '被开方数，不能为负数'
            }
        },
        required: ['value']
    },
    execute: function(params) {
        if (params.value < 0) {
            log.warn(`sqrt called with negative value: ${params.value}`);
            return { error: '负数不能开平方根' };
        }
        const result = Math.sqrt(params.value);
        log.info(`sqrt called: sqrt(${params.value}) = ${result}`);
        return { result: result };
    }
}