document.addEventListener('DOMContentLoaded', () => {
    const taskListUl = document.getElementById('task-list');
    const taskFormContainer = document.getElementById('task-form-container');
    const modal = document.getElementById('modal');
    const modalTitle = document.getElementById('modal-title');
    const modalBody = document.getElementById('modal-body');
    const closeModalBtn = document.querySelector('.close-button');

    let currentTasks = {}; // Store fetched tasks {name: config}
    let selectedTaskName = null; // Keep track of the currently selected task

    // --- API Helper Functions ---

    async function apiFetch(url, options = {}) {
        const response = await fetch(url, options);
        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`HTTP error! status: ${response.status}, message: ${errorText}`);
        }
        const contentType = response.headers.get("content-type");
        return contentType?.includes("application/json")
            ? await response.json()
            : await response.text();
    }

    // --- Task List Management ---

    async function loadTasks() {
        currentTasks = await apiFetch('/api/tasks').catch(() => {
            taskListUl.innerHTML = '<li>Failed to load tasks.</li>';
            return {};
        });
        renderTaskList();
        clearTaskDetailPanel();
    }

    function renderTaskList() {
        taskListUl.innerHTML = '';
        const sortedTaskNames = Object.keys(currentTasks).sort();

        sortedTaskNames.forEach(taskName => {
            const li = document.createElement('li');
            const button = document.createElement('button');
            button.textContent = taskName;
            button.classList.add('task-button');
            button.dataset.taskName = taskName;
            if (taskName === selectedTaskName) {
                button.classList.add('active');
            }
            if (currentTasks[taskName].isNew) {
                button.classList.add('new-task');
            }
            if (currentTasks[taskName].isModified) {
                button.classList.add('modified-task');
            }
            button.addEventListener('click', () => selectTask(taskName));
            li.appendChild(button);
            taskListUl.appendChild(li);
        });

        const addLi = document.createElement('li');
        addLi.classList.add('add-task-item');
        const addButton = document.createElement('button');
        addButton.id = 'add-task-btn';
        addButton.textContent = '+';
        addButton.classList.add('button', 'task-button', 'add-button');
        addButton.addEventListener('click', showNewTaskForm);
        addLi.appendChild(addButton);
        taskListUl.appendChild(addLi);
    }

    function selectTask(taskName) {
        selectedTaskName = taskName;
        renderTaskList(); // Re-render list to highlight the active task
        renderTaskDetail(taskName);
    }

    function clearTaskDetailPanel() {
        taskFormContainer.innerHTML = '<p>Please select a task from the list or add a new one.</p>';
        selectedTaskName = null;
        renderTaskList();
    }

    // --- Helper to get current task context ---
    function getCurrentTaskContext() {
        const form = document.getElementById('task-form');
        if (!form) return null;

        const taskName = form.dataset.taskName;
        const isNew = form.dataset.isNew === 'true';

        if (!taskName && !isNew) {
            alert('Internal Error: Task context is missing. Please select a task or start adding a new one.');
            return null;
        }

        // Ensure the task exists in currentTasks, especially for non-new tasks
        if (!currentTasks[taskName] && !isNew) {
            alert(`Internal Error: Configuration for task "${taskName}" not found.`);
            return null;
        }
        // If it's a new task and doesn't exist yet (e.g., during initial form setup), create a placeholder
        if (!currentTasks[taskName] && isNew) {
            currentTasks[taskName] = { isNew: true }; // Basic structure
        }


        const taskConfig = currentTasks[taskName];


        return { taskName, isNew, taskConfig };
    }


    // --- Task Detail Rendering ---

    function renderTaskDetail(taskName) {
        const taskConfig = currentTasks[taskName];
        if (!taskConfig) {
            clearTaskDetailPanel();
            alert(`Task "${taskName}" not found in currentTasks.`);
            return;
        }

        taskFormContainer.innerHTML = '';

        const form = document.createElement('form');
        form.id = 'task-form';
        form.dataset.taskName = taskName;
        form.dataset.isNew = taskConfig.isNew;

        // --- Basic Info ---
        const infoSection = createFormSection(form, 'Basic Info');
        createTextField(infoSection, 'taskNameDisplay', 'Task Name', taskName, true);
        createNumberField(infoSection, 'interval', 'Fetch Interval (minutes)', taskConfig.interval || 10);

        // --- Downloaders ---
        renderDownloaderSection(form, taskConfig.downloaders || []);

        // --- Feed URLs ---
        renderFeedSection(form, taskConfig.feed?.URLs || []);

        // --- Filter ---
        renderFilterSection(form, taskConfig.filter);

        // --- Extracter ---
        renderExtracterSection(form, taskConfig.extracter);


        // --- Action Buttons ---
        const actionDiv = document.createElement('div');
        actionDiv.classList.add('action-buttons');
        const saveButton = document.createElement('button');
        saveButton.type = 'submit';
        saveButton.style.display = "none";
        saveButton.textContent = form.dataset.isNew === "true" ? 'Create Task' : 'Save Changes';
        saveButton.classList.add('button', 'primary-button');
        if (form.dataset.isNew === "true" || taskConfig.isModified) {
            saveButton.style.display = "block";
        } else {
            saveButton.style.display = "none";
        }
        actionDiv.appendChild(saveButton);

        const deleteButton = document.createElement('button');
        deleteButton.type = 'button'; // Important: prevent form submission
        deleteButton.textContent = 'Delete Task';
        deleteButton.classList.add('button', 'danger-button');
        deleteButton.addEventListener('click', () => deleteTask(taskName));
        actionDiv.appendChild(deleteButton);

        form.appendChild(actionDiv);
        taskFormContainer.appendChild(form);

        form.addEventListener('submit', handleFormSubmit);
    }

    // --- Form Element Creation Helpers ---

    function createFormSection(parent, title) {
        const section = document.createElement('div');
        section.classList.add('form-section');
        const h3 = document.createElement('h3');
        h3.textContent = title;
        section.appendChild(h3);
        parent.appendChild(section);
        return section;
    }

    function createInputField(parent, id, label, options = {}) {
        const { type = 'text', value = '', readOnly = false, min, placeholder } = options;
        const group = document.createElement('div');
        group.classList.add('form-group');

        const labelEl = document.createElement('label');
        labelEl.htmlFor = id;
        labelEl.textContent = label;

        const input = document.createElement('input');
        input.type = type;
        input.id = id;
        input.name = id;
        input.value = value;
        input.readOnly = readOnly;

        if (min !== undefined) input.min = min;
        if (placeholder !== undefined) input.placeholder = placeholder;
        if (readOnly) input.style.backgroundColor = '#eee';

        group.appendChild(labelEl);
        group.appendChild(input);
        parent.appendChild(group);
        return input;
    }

    function createTextField(parent, id, label, value = '', readOnly = false, placeholder) {
        return createInputField(parent, id, label, { type: 'text', value, readOnly, placeholder });
    }

    function createPasswordField(parent, id, label, value = '') {
        return createInputField(parent, id, label, { type: 'password', value });
    }

    function createNumberField(parent, id, label, value = '', readOnly = false, placeholder) {
        return createInputField(parent, id, label, { type: 'number', value, readOnly, min: 1, placeholder });
    }

    function createCheckboxField(parent, id, label, checked = false) {
        const group = document.createElement('div');
        group.classList.add('form-group', 'checkbox-group');

        const input = document.createElement('input');
        input.type = 'checkbox';
        input.id = id;
        input.name = id;
        input.checked = checked;

        const labelEl = document.createElement('label');
        labelEl.htmlFor = id;
        labelEl.textContent = label;

        group.append(input, labelEl);
        parent.appendChild(group);
        return input;
    }

    function createSelectField(parent, id, label, options, selectedValue = '') {
        const group = document.createElement('div');
        group.classList.add('form-group');

        const labelEl = document.createElement('label');
        labelEl.htmlFor = id;
        labelEl.textContent = label;

        const select = document.createElement('select');
        select.id = id;
        select.name = id;

        options.forEach(opt => {
            const optionEl = document.createElement('option');
            optionEl.value = opt.value;
            optionEl.textContent = opt.text;
            optionEl.selected = opt.value === selectedValue;
            select.appendChild(optionEl);
        });

        group.append(labelEl, select);
        parent.appendChild(group);
        return select;
    }

    function createListSection(parent, title, items, renderItemFunc, addItemFunc, listId) {
        const section = createFormSection(parent, title);
        const ul = document.createElement('ul');
        ul.id = listId;
        ul.classList.add('list-items');
        items.forEach((item, index) => renderItemFunc(ul, item, index));
        section.appendChild(ul);

        // Add drag and drop listeners to the list itself if it's draggable
        if (listId === 'downloader-list' || listId === 'feed-url-list') {
            ul.addEventListener('dragover', handleDragOver);
            ul.addEventListener('drop', handleDrop);
            ul.addEventListener('dragleave', handleDragLeave);
        }

        const addButton = document.createElement('button');
        addButton.type = 'button';
        addButton.textContent = `Add ${title.replace(' List', '').replace(' URLs', ' URL')}`; // Make button text more specific
        addButton.classList.add('button', 'secondary-button', 'add-item-button');
        addButton.addEventListener('click', () => addItemFunc(ul)); // Pass the UL element to the add function
        section.appendChild(addButton);
        return ul;
    }


    // --- Section Rendering Functions ---

    function renderDownloaderSection(form, downloaders) {
        const listId = 'downloader-list';
        createListSection(form, 'Downloaders', downloaders, renderDownloaderItem, addDownloader, listId);
    }

    function getRpcUrl(downloader) {
        const defaultPorts = {
            aria2c: 6800,
            transmission: 9091
        };

        const defaultRpcPaths = {
            aria2c: '/jsonrpc',
            transmission: '/transmission/rpc'
        };

        const protocol = downloader.useHttps ? 'https://' : 'http://';
        const port = downloader.port || defaultPorts[downloader.type] || ''; // Default to empty if no type match
        const rpcPath = downloader.rpcPath || defaultRpcPaths[downloader.type] || ''; // Default to empty

        return `${protocol}${downloader.host || 'localhost'}${port ? ':' + port : ''}${rpcPath}`;
    }

    function renderDownloaderItem(ul, downloader, index) {
        const li = document.createElement('li');
        li.dataset.index = index;
        li.dataset.itemType = 'downloader'; // Identify item type
        li.draggable = true; // Make it draggable
        li.classList.add('draggable-item');
        li.addEventListener('dragstart', handleDragStart);
        li.addEventListener('dragend', handleDragEnd);
        li.innerHTML = `
            <span><span class="drag-handle">::</span> <strong>Type:</strong> ${downloader.type} | <strong>RPC URL:</strong> ${getRpcUrl(downloader)}</span>
            <div class="list-item-actions">
                <button type="button" class="edit-downloader-btn button secondary-button">Edit</button>
                <button type="button" class="delete-downloader-btn button danger-button">Del</button>
            </div>
        `;
        li.querySelector('.edit-downloader-btn').addEventListener('click', () => editDownloader(index));
        li.querySelector('.delete-downloader-btn').addEventListener('click', () => deleteDownloader(index));
        ul.appendChild(li);
    }

    function addDownloader() { // Doesn't need ul passed anymore
        openDownloaderModal(); // Open modal to add new
    }

    function editDownloader(index) {
        // Edit Downloader handler
        const context = getCurrentTaskContext();
        if (!context) return;
        const { taskName, taskConfig } = context;
        const downloaderData = taskConfig?.downloaders?.[index];
        if (downloaderData) {
            openDownloaderModal(downloaderData, index);
        } else {
            alert("Could not find downloader data to edit.");
            console.error(`editDownloader: Index ${index} out of bounds or downloaders array missing for task ${taskName}`);
        }
    }

    function deleteDownloader(index) {
        // Delete Downloader handler
        deleteTaskListItem('downloaders', index, `Are you sure you want to delete downloader #${index + 1}?`);
    }


    function renderFeedSection(form, urls) {
        const listId = 'feed-url-list';
        createListSection(form, 'Feed URLs', urls, renderFeedUrlItem, addFeedUrlToList, listId); // Changed add function
    }

    function renderFeedUrlItem(ul, url, index) {
        const li = document.createElement('li');
        li.dataset.index = index;
        li.dataset.itemType = 'feed'; // Identify item type
        li.draggable = true; // Make it draggable
        li.classList.add('draggable-item');
        li.addEventListener('dragstart', handleDragStart);
        li.addEventListener('dragend', handleDragEnd);
        li.innerHTML = `
            <span><span class="drag-handle">::</span> ${url}</span>
            <div class="list-item-actions">
                 <button type="button" class="delete-feed-url-btn button danger-button">Del</button>
            </div>
        `;
        li.querySelector('.delete-feed-url-btn').addEventListener('click', () => deleteFeedUrl(index));
        ul.appendChild(li);
    }

    // Handles adding a feed URL via a modal
    function addFeedUrlToList() {
        openModal("Add Feed URL", (body) => {
            const form = document.createElement('form');
            const input = createTextField(form, 'newFeedUrlInput', 'Feed URL', '', false, 'https://example.com/feed.xml');
            input.required = true; // Basic HTML5 validation

            const errorDiv = document.createElement('div');
            errorDiv.style.color = 'red';
            errorDiv.style.marginTop = '5px';
            form.appendChild(errorDiv);

            const actionDiv = document.createElement('div');
            actionDiv.classList.add('action-buttons');
            const addButton = document.createElement('button');
            addButton.type = 'submit';
            addButton.textContent = 'Add URL';
            addButton.classList.add('button', 'primary-button');
            actionDiv.appendChild(addButton);
            form.appendChild(actionDiv);

            form.addEventListener('submit', (e) => {
                e.preventDefault();
                errorDiv.textContent = ''; // Clear previous errors
                const newUrl = input.value.trim();

                if (newUrl) {
                    const context = getCurrentTaskContext();
                    if (!context) {
                        closeModal();
                        return;
                    }
                    const { taskConfig } = context;

                    // Ensure the path exists before adding
                    if (!taskConfig.feed) {
                        taskConfig.feed = { URLs: [] };
                    }
                    if (!taskConfig.feed.URLs) {
                        taskConfig.feed.URLs = [];
                    }
                    addTaskListItem('feed.URLs', newUrl); // Add to data and re-render
                    closeModal();
                } else {
                    errorDiv.textContent = 'Please enter a valid URL.';
                }
            });

            body.appendChild(form);
            // Set focus to the input field when the modal opens
            setTimeout(() => input.focus(), 0);
        });
    }


    function deleteFeedUrl(index) {
        // Use the generic delete function
        deleteTaskListItem('feed.URLs', index, 'Are you sure you want to delete this feed URL?');
    }


    function renderFilterSection(form, filter) {
        const section = createFormSection(form, 'Filter');
        if (!filter) {
            const addButton = document.createElement('button');
            addButton.type = 'button';
            addButton.textContent = 'Add Filter';
            addButton.classList.add('button', 'secondary-button');
            addButton.addEventListener('click', () => {
                // Use generic toggle function
                toggleTaskSection('filter', { include: [], exclude: [] });
            });
            section.appendChild(addButton);
            return;
        }

        renderKeywordList(section, 'Include Keywords', filter.include || [], 'include-keywords', addIncludeKeyword, deleteIncludeKeyword);
        renderKeywordList(section, 'Exclude Keywords', filter.exclude || [], 'exclude-keywords', addExcludeKeyword, deleteExcludeKeyword);

        const removeFilterBtn = document.createElement('button');
        removeFilterBtn.type = 'button';
        removeFilterBtn.textContent = 'Remove Filter Section';
        removeFilterBtn.classList.add('button', 'danger-button');
        removeFilterBtn.addEventListener('click', () => {
            toggleTaskSection('filter', null, 'Are you sure you want to remove the entire filter section?');
        });
        section.appendChild(removeFilterBtn);
    }

    function renderKeywordList(parent, title, keywords, listId, addFunc, deleteFunc) {
        const subSection = document.createElement('div');
        subSection.classList.add('form-subsection');
        const h4 = document.createElement('h4');
        h4.textContent = title;
        subSection.appendChild(h4);

        const ul = document.createElement('ul');
        ul.id = listId;
        ul.classList.add('list-items'); // Keep class for styling, but not draggable
        keywords.forEach((keyword, index) => renderKeywordItem(ul, keyword, index, deleteFunc));
        subSection.appendChild(ul);

        const addForm = document.createElement('div');
        addForm.classList.add('add-item-form');
        const keywordInput = document.createElement('input');
        keywordInput.type = 'text';
        keywordInput.placeholder = 'Enter new keyword(s)';
        const addButton = document.createElement('button');
        addButton.type = 'button';
        addButton.textContent = 'Add';
        addButton.classList.add('button', 'secondary-button');
        addButton.addEventListener('click', () => {
            const newKeyword = keywordInput.value.trim();
            if (newKeyword) {
                addFunc(newKeyword); // Pass only the keyword to the add function
                keywordInput.value = '';
            } else {
                alert('Please enter a keyword.');
            }
        });
        addForm.appendChild(keywordInput);
        addForm.appendChild(addButton);
        subSection.appendChild(addForm);

        parent.appendChild(subSection);
    }

    function renderKeywordItem(ul, keyword, index, deleteFunc) {
        const li = document.createElement('li');
        li.dataset.index = index; // Store index for deletion
        li.innerHTML = `
            <span>${keyword}</span>
            <div class="list-item-actions">
                 <button type="button" class="delete-keyword-btn button danger-button">Del</button>
            </div>
        `;
        li.querySelector('.delete-keyword-btn').addEventListener('click', () => deleteFunc(index));
        ul.appendChild(li);
    }

    function addIncludeKeyword(keyword) {
        addTaskListItem('filter.include', keyword);
    }

    function deleteIncludeKeyword(index) {
        deleteTaskListItem('filter.include', index, 'Are you sure you want to delete this include keyword?');
    }

    function addExcludeKeyword(keyword) {
        addTaskListItem('filter.exclude', keyword);
    }

    function deleteExcludeKeyword(index) {
        deleteTaskListItem('filter.exclude', index, 'Are you sure you want to delete this exclude keyword?');
    }


    function renderExtracterSection(form, extracter) {
        const section = createFormSection(form, 'Extracter');
        if (!extracter) {
            const addButton = document.createElement('button');
            addButton.type = 'button';
            addButton.textContent = 'Add Extracter';
            addButton.classList.add('button', 'secondary-button');
            addButton.addEventListener('click', () => {
                toggleTaskSection('extracter', { tag: 'link', pattern: '' });
            });
            section.appendChild(addButton);
            return;
        }

        const validTags = ['title', 'link', 'description', 'enclosure', 'guid'];
        const tagOptions = validTags.map(tag => ({ value: tag, text: tag }));
        createSelectField(section, 'extracterTag', 'Tag', tagOptions, extracter.tag);
        createTextField(section, 'extracterPattern', 'Pattern (Regex)', extracter.pattern || '');

        const removeExtracterBtn = document.createElement('button');
        removeExtracterBtn.type = 'button';
        removeExtracterBtn.textContent = 'Remove Extracter Section';
        removeExtracterBtn.classList.add('button', 'danger-button');
        removeExtracterBtn.id = "removeExtracterBtn";
        removeExtracterBtn.addEventListener('click', () => {
            toggleTaskSection('extracter', null, 'Are you sure you want to remove the extracter section?');
        });
        section.appendChild(removeExtracterBtn);
    }

    // --- Generic Data Manipulation Helpers ---

    // Helper to safely get a nested property
    function getNestedProperty(obj, path) {
        if (!obj || !path) return undefined;
        const parts = path.split('.');
        let current = obj;
        for (const part of parts) {
            if (current === null || current === undefined) return undefined;
            current = current[part];
        }
        return current;
    }

    // Helper to safely set a nested property, creating objects/arrays if needed
    function setNestedProperty(obj, path, value) {
        if (!obj || !path) return;
        const parts = path.split('.');
        let current = obj;
        for (let i = 0; i < parts.length - 1; i++) {
            const part = parts[i];
            if (current[part] === undefined || current[part] === null) {
                // Look ahead to see if next part implies an array index
                const nextPart = parts[i + 1];
                current[part] = /^\d+$/.test(nextPart) ? [] : {};
            }
            current = current[part];
        }
        current[parts[parts.length - 1]] = value;
    }

    // Generic function to add an item to a list within the current task's config
    function addTaskListItem(itemPath, value) {
        const context = getCurrentTaskContext();
        if (!context) return;
        const { taskName, taskConfig } = context;

        const itemsArray = getNestedProperty(taskConfig, itemPath);
        if (!Array.isArray(itemsArray)) {
            console.error(`addTaskListItem: Path "${itemPath}" does not resolve to an array in task "${taskName}".`);
            // Attempt to initialize it if possible? Or just error out.
            // Let's try initializing if the parent exists.
            const parentPath = itemPath.substring(0, itemPath.lastIndexOf('.'));
            const parentObj = parentPath ? getNestedProperty(taskConfig, parentPath) : taskConfig;
            if (parentObj) {
                const arrayName = itemPath.substring(itemPath.lastIndexOf('.') + 1);
                parentObj[arrayName] = []; // Initialize as empty array
                console.log(`Initialized array at path: ${itemPath}`);
            } else {
                console.error(`Cannot initialize array, parent path "${parentPath}" not found.`);
                return;
            }
            // Re-fetch the (now initialized) array
            const newItemsArray = getNestedProperty(taskConfig, itemPath);
            if (!Array.isArray(newItemsArray)) {
                console.error("Failed to initialize array.");
                return;
            }
            newItemsArray.push(value);
        } else {
            itemsArray.push(value);
        }


        console.log(`Added item to ${itemPath}:`, value);
        // Re-render the whole task detail to reflect the change
        renderTaskDetail(taskName);
        // Mark task as modified
        if (!currentTasks[taskName].isNew) {
            currentTasks[taskName].isModified = true;
            renderTaskList();
        }
        const saveButton = document.querySelector('#task-form .primary-button');
        if (saveButton) {
            saveButton.style.display = "block";
        }
    }

    // Generic function to delete an item from a list within the current task's config
    function deleteTaskListItem(itemPath, index, confirmMessage) {
        if (!confirm(confirmMessage)) return;

        const context = getCurrentTaskContext();
        if (!context) return;
        const { taskName, taskConfig } = context;

        const itemsArray = getNestedProperty(taskConfig, itemPath);
        if (!Array.isArray(itemsArray)) {
            console.error(`deleteTaskListItem: Path "${itemPath}" does not resolve to an array in task "${taskName}".`);
            return;
        }

        if (index >= 0 && index < itemsArray.length) {
            itemsArray.splice(index, 1);
            console.log(`Deleted item at index ${index} from ${itemPath}`);
            // Re-render the whole task detail
            renderTaskDetail(taskName);
            // Mark task as modified
            if (!currentTasks[taskName].isNew) {
                currentTasks[taskName].isModified = true;
                renderTaskList();
            }
            const saveButton = document.querySelector('#task-form .primary-button');
            if (saveButton) {
                saveButton.style.display = "block";
            }
        } else {
            console.error(`deleteTaskListItem: Invalid index ${index} for path "${itemPath}" in task "${taskName}".`);
        }
    }

    // Generic function to add/remove an entire section (object) from the task config
    function toggleTaskSection(sectionName, defaultData = null, confirmMessage = null) {
        if (confirmMessage && !confirm(confirmMessage)) return;

        const context = getCurrentTaskContext();
        if (!context) return;
        const { taskName, taskConfig } = context;

        if (taskConfig[sectionName] === undefined || taskConfig[sectionName] === null) {
            // Add the section
            taskConfig[sectionName] = defaultData;
            console.log(`Added section "${sectionName}"`);
        } else {
            // Remove the section
            delete taskConfig[sectionName];
            console.log(`Removed section "${sectionName}"`);
        }
        // Re-render the detail panel
        renderTaskDetail(taskName);
        // Mark task as modified
        if (!currentTasks[taskName].isNew) {
            currentTasks[taskName].isModified = true;
            renderTaskList();
        }
        const saveButton = document.querySelector('#task-form .primary-button');
        if (saveButton) {
            saveButton.style.display = "block";
        }
    }


    // --- Form Submission and Deletion ---

    async function handleFormSubmit(event) { // Keep async as it calls apiFetch
        event.preventDefault();
        const form = event.target;
        const taskName = form.dataset.taskName;
        const isNew = form.dataset.isNew === 'true';

        const context = getCurrentTaskContext(); // Get current data
        if (!context) return; // Should not happen if form exists, but check anyway
        const { taskConfig: currentConfig } = context;

        // Construct the final task config from the form and existing data
        const finalTaskConfig = {
            // Start with existing non-form-related data if editing
            ...(isNew ? {} : currentConfig),
            // Overwrite with form values
            interval: parseInt(form.elements.interval.value, 10) || 10,
            // Downloaders and Feed URLs are managed by their respective add/delete/drag functions
            // directly modifying currentTasks[taskName].downloaders and currentTasks[taskName].feed.URLs
            downloaders: currentConfig.downloaders || [], // Get potentially reordered array
            feed: {
                URLs: currentConfig.feed?.URLs || [] // Get potentially reordered array
            },
            // Filter section
            filter: undefined, // Start with undefined
            // Extracter section
            extracter: undefined, // Start with undefined
        };

        // Populate Filter if section exists
        const includeList = form.querySelector('#include-keywords');
        const excludeList = form.querySelector('#exclude-keywords');
        if (includeList || excludeList) {
            finalTaskConfig.filter = {
                include: currentConfig.filter?.include || [], // Get potentially modified array
                exclude: currentConfig.filter?.exclude || []
            };
        }
        // Clean up Filter: remove empty include/exclude arrays and the filter object itself if empty
        if (finalTaskConfig.filter) {
            if (finalTaskConfig.filter.include?.length === 0) delete finalTaskConfig.filter.include;
            if (finalTaskConfig.filter.exclude?.length === 0) delete finalTaskConfig.filter.exclude;
            if (Object.keys(finalTaskConfig.filter).length === 0) delete finalTaskConfig.filter;
        }

        // Update Extracter directly from form fields if the section exists
        const extracterTagEl = form.querySelector('#extracterTag');
        const extracterPatternEl = form.querySelector('#extracterPattern');
        if (extracterTagEl && extracterPatternEl) {
            const pattern = extracterPatternEl.value.trim();
            if (pattern) {
                finalTaskConfig.extracter = {
                    tag: extracterTagEl.value,
                    pattern: pattern
                };
            }
        }

        // --- Basic Validation (using finalTaskConfig) ---
        if (!finalTaskConfig.feed?.URLs?.length) {
            alert('Task must have at least one feed URL.');
            return;
        }
        if (!finalTaskConfig.downloaders?.length) {
            alert('Task must have at least one downloader.');
            return;
        }

        console.log("Submitting task data:", taskName, finalTaskConfig);

        try {
            let result;
            if (isNew) {
                // Create new task
                result = await apiFetch('/api/tasks', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ name: taskName, config: finalTaskConfig })
                });
                alert(`Task "${taskName}" created successfully!`);
                delete finalTaskConfig.isNew; // Remove the temporary new flag
            } else {
                // Update existing task
                result = await apiFetch(`/api/tasks/${taskName}`, {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(finalTaskConfig)
                });
                alert(`Task "${taskName}" updated successfully!`);
                delete finalTaskConfig.isModified;
            }
            console.log('API Response:', result);
            // Update local state and UI
            currentTasks[taskName] = finalTaskConfig; // Update local cache with saved data
            selectedTaskName = taskName; // Ensure it remains selected
            renderTaskList(); // Update list (e.g., remove new task indicator)
            renderTaskDetail(taskName); 

        } catch (error) {
            console.error('Failed to save task:', error);
            alert(`Error saving task "${taskName}": ${error.message}`);
        }
    }

    async function deleteTask(taskName) {
        if (!confirm(`Are you sure you want to delete the task "${taskName}"?`)) {
            return;
        }

        if (!currentTasks[taskName].isNew) {
            try {
                await apiFetch(`/api/tasks/${taskName}`, { method: 'DELETE' });
                alert(`Task "${taskName}" deleted successfully!`);
            } catch (error) {
                console.error('Failed to delete task:', error);
                alert(`Error deleting task "${taskName}": ${error.message}`);
            }
        }
        // Remove from local state and update UI
        delete currentTasks[taskName];
        clearTaskDetailPanel(); // Clear detail panel
        renderTaskList(); // Update task list
    }

    // --- Add New Task ---

    function showNewTaskForm() {
        openModal("Add New Task", (body) => {
            const form = document.createElement('form');
            const input = createTextField(form, 'newTaskNameInput', 'New Task Name', '', false, 'My New Feed Task');
            input.required = true;

            const errorDiv = document.createElement('div');
            errorDiv.style.color = 'red';
            errorDiv.style.marginTop = '5px';
            form.appendChild(errorDiv);

            const actionDiv = document.createElement('div');
            actionDiv.classList.add('action-buttons');
            const createButton = document.createElement('button');
            createButton.type = 'submit';
            createButton.textContent = 'Create Task';
            createButton.classList.add('button', 'primary-button');
            actionDiv.appendChild(createButton);
            form.appendChild(actionDiv);

            form.addEventListener('submit', (e) => {
                e.preventDefault();
                errorDiv.textContent = ''; // Clear previous errors
                const newTaskName = input.value.trim();

                if (!newTaskName) {
                    errorDiv.textContent = "Task name cannot be empty.";
                    return;
                }
                if (currentTasks[newTaskName]) {
                    errorDiv.textContent = `Task "${newTaskName}" already exists.`;
                    return;
                }

                // Validation passed, proceed with task creation logic
                // Create a temporary placeholder in currentTasks
                currentTasks[newTaskName] = {
                    isNew: true, // Flag to indicate it's a new task
                    interval: 10, // Default interval
                    downloaders: [],
                    feed: { URLs: [] },
                };

                selectedTaskName = newTaskName; // Select the new task
                renderTaskList(); // Update list to show the new task
                renderTaskDetail(newTaskName); // Render the form for the new task
                closeModal(); // Close the modal on success
            });

            body.appendChild(form);
            // Set focus to the input field when the modal opens
            setTimeout(() => input.focus(), 0);
        });
    }

    // --- Modal Management ---

    function openModal(title, contentGenerator) {
        modalTitle.textContent = title;
        modalBody.innerHTML = ''; // Clear previous content
        contentGenerator(modalBody); // Populate modal body
        modal.style.display = 'block';
    }

    function closeModal() {
        modal.style.display = 'none';
        modalBody.innerHTML = ''; // Clear content on close
    }

    // Close modal if clicked outside the content area
    window.onclick = function (event) {
        if (event.target == modal) {
            closeModal();
        }
    }
    // Close modal with the close button
    closeModalBtn.onclick = closeModal;


    // --- Downloader Modal Specific Logic ---

    function openDownloaderModal(downloaderData = null, index = null) {
        const isEditing = downloaderData !== null && index !== null;
        const title = isEditing ? `Edit Downloader #${index + 1}` : 'Add New Downloader';

        openModal(title, (body) => {
            body.innerHTML = ''; // Clear previous content

            const form = document.createElement('form');
            form.id = 'downloader-form';

            // Hidden input for index
            const indexInput = document.createElement('input');
            indexInput.type = 'hidden';
            indexInput.name = 'downloaderIndex';
            indexInput.value = index ?? '';
            form.appendChild(indexInput);

            // Initial values
            const type = downloaderData?.type || 'aria2c';
            const host = downloaderData?.host || 'localhost';
            const useHttps = downloaderData?.useHttps || false;
            const autoCleanUp = downloaderData?.autoCleanUp || false;

            // Type Select
            const downloaderTypeOptions = [
                { value: 'aria2c', text: 'Aria2c' },
                { value: 'transmission', text: 'Transmission' }
            ];
            const typeSelect = createSelectField(form, 'downloaderType', 'Type', downloaderTypeOptions, type);

            // Host
            createTextField(form, 'downloaderHost', 'Host', host);

            // --- Aria2c Specific Fields ---
            const aria2cFieldsDiv = document.createElement('div');
            aria2cFieldsDiv.id = 'aria2c-fields';
            createNumberField(aria2cFieldsDiv, 'downloaderAria2cPort', 'Port (optional)', downloaderData?.port || '', false, 'e.g., 6800');
            createTextField(aria2cFieldsDiv, 'downloaderAria2cRpcPath', 'RPC Path (optional)', downloaderData?.rpcPath || '', false, 'e.g., /jsonrpc');
            createTextField(aria2cFieldsDiv, 'downloaderAria2cToken', 'Token (optional)', downloaderData?.token || '');
            form.appendChild(aria2cFieldsDiv);

            // --- Transmission Specific Fields ---
            const transmissionFieldsDiv = document.createElement('div');
            transmissionFieldsDiv.id = 'transmission-fields';
            createNumberField(transmissionFieldsDiv, 'downloaderTrasnPort', 'Port (optional)', downloaderData?.port || '', false, 'e.g., 9091');
            createTextField(transmissionFieldsDiv, 'downloaderTrasnRpcPath', 'RPC Path (optional)', downloaderData?.rpcPath || '', false, 'e.g., /transmission/rpc');
            createTextField(transmissionFieldsDiv, 'downloaderTrasnUsername', 'Username (optional)', downloaderData?.username || '');
            createPasswordField(transmissionFieldsDiv, 'downloaderTrasnSecret', 'Secret/Password (optional)', downloaderData?.password || '');
            form.appendChild(transmissionFieldsDiv);

            // Common Checkboxes
            createCheckboxField(form, 'downloaderUseHttps', 'Use HTTPS', useHttps);
            createCheckboxField(form, 'downloaderAutoCleanUp', 'Auto CleanUp', autoCleanUp);

            // Action Buttons
            const actionDiv = document.createElement('div');
            actionDiv.classList.add('action-buttons');
            const submitButton = document.createElement('button');
            submitButton.type = 'submit';
            submitButton.textContent = isEditing ? 'Save Downloader' : 'Add Downloader';
            submitButton.classList.add('button', 'primary-button');
            actionDiv.appendChild(submitButton);
            form.appendChild(actionDiv);

            body.appendChild(form); // Add the constructed form to the modal body

            // --- Event Listeners ---

            // Form Submission
            form.addEventListener('submit', (e) => {
                e.preventDefault();
                const formData = new FormData(form); // Use the created form element
                const indexToEdit = formData.get('downloaderIndex') ? parseInt(formData.get('downloaderIndex'), 10) : null;

                const newDownloader = {
                    type: formData.get('downloaderType'),
                    host: formData.get('downloaderHost').trim() || 'localhost',
                    useHttps: formData.get('downloaderUseHttps') === 'on',
                    autoCleanUp: formData.get('downloaderAutoCleanUp') === 'on'
                };

                if (newDownloader.type === 'aria2c') {
                    newDownloader.port = formData.get('downloaderAria2cPort') ? parseInt(formData.get('downloaderAria2cPort'), 10) : undefined;
                    newDownloader.rpcPath = formData.get('downloaderAria2cRpcPath').trim() || undefined;
                    newDownloader.token = formData.get('downloaderAria2cToken').trim() || undefined;
                } else if (newDownloader.type === 'transmission') {
                    newDownloader.port = formData.get('downloaderTrasnPort') ? parseInt(formData.get('downloaderTrasnPort'), 10) : undefined;
                    newDownloader.rpcPath = formData.get('downloaderTrasnRpcPath').trim() || undefined;
                    newDownloader.username = formData.get('downloaderTrasnUsername').trim() || undefined;
                    newDownloader.password = formData.get('downloaderTrasnSecret') || undefined;
                }

                Object.keys(newDownloader).forEach(key => {
                    if (newDownloader[key] === undefined || newDownloader[key] === '') {
                        delete newDownloader[key];
                    }
                });

                const context = getCurrentTaskContext();
                if (!context) { closeModal(); return; } // Ensure modal closes on error
                const { taskName, taskConfig } = context;

                if (!taskConfig.downloaders) {
                    taskConfig.downloaders = [];
                }

                if (indexToEdit !== null && indexToEdit >= 0 && indexToEdit < taskConfig.downloaders.length) {
                    taskConfig.downloaders[indexToEdit] = newDownloader;
                    console.log(`Updated downloader at index ${indexToEdit}`);
                } else {
                    taskConfig.downloaders.push(newDownloader);
                    console.log("Added new downloader");
                }

                closeModal();
                renderTaskDetail(taskName);
                if (!currentTasks[taskName].isNew) {
                    currentTasks[taskName].isModified = true;
                    renderTaskList();
                }
            });

            // Toggle field visibility
            const toggleSpecificFields = () => {
                const selectedType = typeSelect.value; // Use the created typeSelect element
                aria2cFieldsDiv.style.display = selectedType === 'aria2c' ? 'block' : 'none';
                transmissionFieldsDiv.style.display = selectedType === 'transmission' ? 'block' : 'none';
            };
            typeSelect.addEventListener('change', toggleSpecificFields);
            toggleSpecificFields(); // Initial call
        });
    }


    // --- Drag and Drop Handlers ---

    let draggedItem = null; // Keep track of the item being dragged

    function handleDragStart(e) {
        // Ensure the target is the LI element itself
        if (!e.target.classList.contains('draggable-item')) return;
        draggedItem = e.target;
        e.dataTransfer.effectAllowed = 'move';
        // Optional: Store data, though we can get it from draggedItem later
        e.dataTransfer.setData('text/plain', draggedItem.dataset.index);
        // Add a class to the dragged item for visual feedback (defer to avoid issues)
        setTimeout(() => {
            if (draggedItem) draggedItem.classList.add('dragging');
        }, 0);
    }

    function handleDragEnd(e) {
        // Remove the dragging class
        if (draggedItem) {
            draggedItem.classList.remove('dragging');
        }
        // Remove any drag-over class from potential targets
        document.querySelectorAll('.drag-over').forEach(el => el.classList.remove('drag-over'));
        draggedItem = null;
    }

    function handleDragOver(e) {
        e.preventDefault(); // Necessary to allow dropping
        if (!draggedItem) return; // Only act if an item is being dragged

        const targetList = e.currentTarget; // The UL element
        // Find the closest LI element that is draggable, could be the target itself or a parent
        const targetItem = e.target.closest('li.draggable-item');

        // Ensure we are dragging over a valid list for the item type
        const listId = targetList.id;
        const itemType = draggedItem.dataset.itemType;
        if ((listId === 'downloader-list' && itemType !== 'downloader') ||
            (listId === 'feed-url-list' && itemType !== 'feed')) {
            e.dataTransfer.dropEffect = 'none'; // Indicate invalid drop target
            // Clean up any lingering highlights from other lists
            document.querySelectorAll(`#${listId} .drag-over`).forEach(el => el.classList.remove('drag-over'));
            return;
        }

        e.dataTransfer.dropEffect = 'move'; // Indicate valid drop target

        // Remove previous drag-over highlights from siblings within the *same* list
        Array.from(targetList.children).forEach(child => {
            if (child !== targetItem && child.classList.contains('draggable-item')) { // Only check draggable items
                child.classList.remove('drag-over');
            }
        });

        if (targetItem && targetItem !== draggedItem) {
            targetItem.classList.add('drag-over'); // Highlight the potential drop target
        }
    }

    function handleDragLeave(e) {
        // Remove drag-over class when leaving a potential target LI or the list itself
        const currentTarget = e.currentTarget; // The UL
        const relatedTarget = e.relatedTarget; // Where the mouse is going

        // Check if the mouse is leaving the list container entirely
        if (!currentTarget.contains(relatedTarget)) {
            // Remove highlight from all items in this list
            Array.from(currentTarget.children).forEach(child => {
                child.classList.remove('drag-over');
            });
        } else if (e.target.classList.contains('draggable-item') && e.target !== relatedTarget?.closest('li.draggable-item')) {
            // If leaving a specific LI, remove its highlight, unless moving directly to another LI
            e.target.classList.remove('drag-over');
        }
    }

    function handleDrop(e) {
        e.preventDefault();
        e.stopPropagation(); // Prevent event bubbling

        if (!draggedItem) return;

        const targetList = e.currentTarget; // The UL element
        const targetItem = e.target.closest('li.draggable-item'); // The LI being dropped onto/near
        const listId = targetList.id; // e.g., 'downloader-list' or 'feed-url-list'
        const itemType = draggedItem.dataset.itemType; // 'downloader' or 'feed'

        // Clean up visual cues immediately
        document.querySelectorAll('.drag-over').forEach(el => el.classList.remove('drag-over'));
        if (draggedItem) draggedItem.classList.remove('dragging');


        // Ensure drop is within the same list type
        if ((listId === 'downloader-list' && itemType !== 'downloader') ||
            (listId === 'feed-url-list' && itemType !== 'feed')) {
            console.warn("Cannot drop item into a list of a different type.");
            draggedItem = null; // Reset dragged item
            return;
        }

        // Don't drop on itself
        if (targetItem === draggedItem) {
            draggedItem = null; // Reset dragged item
            return;
        }

        const context = getCurrentTaskContext();
        if (!context) {
            draggedItem = null; return;
        }
        const { taskName, taskConfig } = context;

        let itemsArray;
        let arrayPath; // Path to the array within taskConfig

        if (itemType === 'downloader') {
            arrayPath = 'downloaders';
            if (!taskConfig.downloaders) taskConfig.downloaders = []; // Ensure array exists
            itemsArray = taskConfig.downloaders;
        } else if (itemType === 'feed') {
            arrayPath = 'feed.URLs';
            // Ensure feed and URLs array exist
            if (!taskConfig.feed) taskConfig.feed = { URLs: [] };
            if (!taskConfig.feed.URLs) taskConfig.feed.URLs = [];
            itemsArray = taskConfig.feed.URLs;
        } else {
            console.error("Unknown draggable item type:", itemType);
            draggedItem = null; return;
        }

        if (!itemsArray) {
            console.error(`Could not find items array for type "${itemType}" in task "${taskName}"`);
            draggedItem = null; return;
        }

        const draggedIndex = parseInt(draggedItem.dataset.index, 10);
        // Check if index is valid
        if (isNaN(draggedIndex) || draggedIndex < 0 || draggedIndex >= itemsArray.length) {
            console.error("Invalid dragged item index:", draggedItem.dataset.index);
            draggedItem = null; return;
        }
        const itemToMove = itemsArray[draggedIndex];

        // Remove item from its original position in the data array
        itemsArray.splice(draggedIndex, 1);

        let targetIndex;
        if (targetItem) {
            // Dropped onto another item
            // Get the index of the target item *in the current DOM order* before inserting
            const currentDomItems = Array.from(targetList.children);
            targetIndex = currentDomItems.indexOf(targetItem);

            // Insert the moved item at this index in the data array
            itemsArray.splice(targetIndex, 0, itemToMove);
        } else {
            // Dropped onto the list but not onto a specific item (append to end)
            itemsArray.push(itemToMove);
            targetIndex = itemsArray.length - 1; // New index is the last one
        }

        console.log(`Moved ${itemType} from original index ${draggedIndex} to new index ${targetIndex}`);

        // --- Re-render the list ---

        // Clear the current list in the DOM
        targetList.innerHTML = '';
        // Re-populate the list based on the updated itemsArray, assigning new indices
        itemsArray.forEach((item, index) => {
            if (itemType === 'downloader') {
                renderDownloaderItem(targetList, item, index);
            } else if (itemType === 'feed') {
                renderFeedUrlItem(targetList, item, index);
            }
        });

        // Mark task as modified
        if (!currentTasks[taskName].isNew) {
            currentTasks[taskName].isModified = true;
            renderTaskList();
        }
        const saveButton = document.querySelector('#task-form .primary-button');
        if (saveButton) {
            saveButton.style.display = "block";
        }

        draggedItem = null; // Reset dragged item state
    }


    // --- Initial Load ---
    loadTasks();
});