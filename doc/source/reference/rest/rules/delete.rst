Supprimer une règle
===================

.. http:delete:: /api/rules/(string:rule_name)

   Supprime la règle demandée.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 204: La règle a été supprimée avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: La règle demandée n'existe pas

   **Exemple de requête**

   .. code-block:: http

      DELETE https://my_waarp_gateway.net/api/rules/règle_1 HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 204 NO CONTENT
