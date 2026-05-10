module.exports = {
    type: 'tool',
    name: 'subtract',
    description: '两个数字相减',
    inputSchema: {
        type: 'object',
        properties: {
            minuend: {
                type: 'number',
                description: '被减数'
            },
            subtrahend: {
                type: 'number',
                description: '减数'
            }
        },
        required: ['minuend', 'subtrahend']
    },
    execute: function(params) {
        return { result: params.minuend - params.subtrahend };
    }
}