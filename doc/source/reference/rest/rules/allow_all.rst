####################################
Enlever les restrictions d'une règle
####################################

.. http:put:: /api/rules/(string:rule_name)/allow_all

   Supprime toutes les restrictions d'utilisation imposées sur la règle, la
   rendant, de fait, utilisable par tous. Pour supprimer une permission en
   particulier, se référer aux chapitres :

   - :any:`reference-rest-servers-revoke`
   - :any:`reference-rest-servers-accounts-revoke`
   - :any:`reference-rest-partners-revoke`
   - :any:`reference-rest-partners-accounts-revoke`

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 200: La règle a été supprimée avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: La règle demandée n'existe pas

   **Exemple de requête**

   .. code-block:: http

      DELETE https://my_waarp_gateway.net/api/rules/règle_1 HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 200 OK
