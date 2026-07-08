==============================
Déclencher un filewatcher
==============================

.. program:: waarp-gateway filewatcher fire

Déclenche immédiatement une exécution du *filewatcher* donné en paramètre,
sans attendre la prochaine occurrence de son intervalle.

**Commande**

.. code-block:: shell

   waarp-gateway filewatcher fire "<FLOW>"

**Exemple**

.. code-block:: shell

   waarp-gateway filewatcher fire 'my-filewatcher'
