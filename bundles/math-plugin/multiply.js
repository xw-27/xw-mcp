module.exports = {
    type: 'tool',
    name: 'multiply',
    description: '两个数字相乘',
    inputSchema: {
        type: 'object',
        properties: {
            multiplicand: {
                type: 'number',
                description: '被乘数'
            },
            multiplier: {
                type: 'number',
                description: '乘数'
            }
        },
        required: ['multiplicand', 'multiplier']
    },
    execute: function(params) {
        return { result: params.multiplicand * params.multiplier };
    }
}