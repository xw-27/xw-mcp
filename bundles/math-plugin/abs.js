module.exports = {
    type: 'tool',
    name: 'abs',
    description: '计算绝对值',
    inputSchema: {
        type: 'object',
        properties: {
            value: {
                type: 'number',
                description: '待求绝对值的数字'
            }
        },
        required: ['value']
    },
    execute: function(params) {
        return { result: Math.abs(params.value) };
    }
}