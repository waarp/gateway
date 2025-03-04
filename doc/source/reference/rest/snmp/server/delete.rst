Supprimer le serveur SNMP
=========================

.. http:delete:: /api/snmp/server

   Supprime le serveur SNMP s'il existe.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 204: Le serveur a été supprimé avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: La gateway n'a pas de serveur SNMP

   **Exemple de requête**

   .. code-block:: http

      DELETE https://my_waarp_gateway.net/api/snmp/server HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 204 NO CONTENT
