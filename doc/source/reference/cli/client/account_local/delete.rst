=========================
Supprimer un compte local
=========================

.. program:: waarp-gateway account local delete

.. describe:: waarp-gateway account local <SERVER> delete <LOGIN>

Supprime le compte donné en paramètre de commande.

|

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'http://user:password@localhost:8080' account local 'serveur_sftp' delete 'tata'