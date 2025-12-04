eino官网地址:https://www.cloudwego.io/zh/docs/eino/

# agent: reactAgent、workflowAgents、multiAgent

# reactAgent:
模型和工具结合,是一个智能决策的大脑,常用场景知识问答、IT运维(可根据线索一步步定位问题)

# workflowAgents: 
定制化流水线,按照顺序严格执行,常用场景比如CICD、数据迁移等

# multiAgent:
一种是集中式协调(Supervisor),中心化设计,主agent管理一批子agent,主agent可以根据情况来进行动态调整和任务分配给子agent;
另外一种是结构化问题解决( Plan-Execute),


^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^各组件定义和使用

---------------------------------------------------------------Graph 编排
一、Graph 编排
1、使用时
将各个组件先添加到graph,然后通过AddEdge将各个组件串联起来,首尾分别是:compose.START和compose.END,其他组件(名称)在中间,
并且按照头尾相连.
各组件包括:AddChatTemplateNode、AddChatModelNode、AddToolsNode、
AddLambdaNode、AddRetrieverNode、AddEdge、AddBranch、AddDocumentTransformerNode、AddEmbeddingNode、AddIndexerNode等

2、全局状态

---------------------------------------------------------------Graph 编排

