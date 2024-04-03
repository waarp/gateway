========================================
Supprimer une méthode d'authentification
========================================

.. program:: waarp-gateway account local auth delete

.. describe:: waarp-gateway account local <SERVER> auth <LOGIN> delete

Supprime la valeur d'authentification données du compte.

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'http://user:password@localhost:8080' account local 'gw_r66' auth 'tata' delete 'r66_password'
