====================================
Enlever les restrictions d'une règle
====================================

.. program:: waarp-gateway rule delete

Enlève toutes les restrictions d'utilisation imposées sur une règle. La
règle devient donc utilisable par tous les agents connus. Le nom de la
règle être spécifié en argument de programme.

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'http://user:password@localhost:8080' rule allow 'règle_1'
