module.exports = {
    type: 'tool',
    name: 'power',
    description: '计算幂运算',
    inputSchema: {
        type: 'object',
        properties: {
            base: {
                type: 'number',
                description: '底数'
            },
            exponent: {
                type: 'number',
                description: '指数'
            }
        },
        required: ['base', 'exponent']
    },
    execute: function(params) {
        const result = Math.pow(params.base, params.exponent);
        log.info(`power called: ${params.base} ^ ${params.exponent} = ${result}`);
        return { result: result };
    }
}