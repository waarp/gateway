Supprimer un moniteur
=====================

.. http:delete:: /api/snmp/monitors/{string:monitor}

   Supprime le moniteur SNMP demandé.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 204: Le moniteur a été supprimé avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le moniteur demandé n'existe pas

   **Exemple de requête**

   .. code-block:: http

      DELETE https://my_waarp_gateway.net/api/snmp/monitors/snmp-monitor HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 204 NO CONTENT
